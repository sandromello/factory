package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"os"

	"io/ioutil"
	"strings"

	"github.com/golang/glog"
	"github.com/sandromello/factory/pkg/conf"
	"github.com/sandromello/factory/pkg/git"
	"github.com/sandromello/factory/pkg/pack"
	"github.com/sandromello/factory/pkg/version"
	"github.com/spf13/pflag"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

var (
	cfg         conf.Config
	showVersion bool
)

const (
	secretsPath = "/var/run/fkt"
	// kubectl create secret docker-registry <name> --docker-server=<registry-fqdn> --docker-username=<user> --docker-password=<pass> --docker-email=<email>
	// {"registry-fqdn":{"username":"<user>","password":"<pass>","email":"<email>","auth":"<base64-auth>"}}
	registrySecretFilePath = secretsPath + "/.dockercfg"
	// A comma separated user, password: <user>,<password>
	gitBasicAuthFilePath = secretsPath + "/.git-basic-auth"
	// A string github oauth token
	gitOauthTokenFilePath = secretsPath + "/.git-oauth-token"
)

// TODO: accept basic and oauth authentication on cloning
// TODO: verify if the repo has a Dockerfile
// TODO: detect types of languages (buildpacks): https://github.com/cloud66/starter
// TODO: parse Procfile and use it as ENTRYPOINT

// - Identify the code language: https://github.com/Azure/draft/issues/205
// - Build
// - Install dependencies packages for each language (python, php, nodejs, ruby, go)
// - Push to registry
// - Parse Procfile and run as command inside the image (Procfile is anallogous to CMD in a Dockefile)
// - Ignore Dockefile of the user*
// - Detect languages and use Dockefile templates

// If contain a Dockefile verifies if it's a platform ONE, validing the `FROM`
// Don't let it run as ROOT!

// Create a Procfile parser for every pack
// Add inside kubernetes (fallback retrieving git and registry credentials inside secrets!)
// Add semantic versioning parser

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.StringVar(&cfg.DockerAddr, "docker-addr", "unix:///var/run/docker.sock", "the address the docker engine listens on")
	pflag.StringVar(&cfg.RegistryOrg, "registry-org", "", "the organization used to push images up to the registry")
	pflag.StringVar(&cfg.RegistryURL, "registry-url", "docker.io", "the URL of the registry (e.g. quay.io, docker.io, gcr.io")
	pflag.BoolVar(&cfg.PushToRegistry, "push", false, "after the build push to the registry")
	pflag.StringVar(&cfg.CloneInfo.URL, "clone-url", "", "the GIT clone URL where the app resides on")
	pflag.StringVar(&cfg.CloneInfo.Path, "clone-path", "", "the path to clone the source app")
	pflag.StringVar(&cfg.CloneInfo.Ref, "git-ref", plumbing.Master.String(), "the git reference (branches only) to clone")
	pflag.StringVar(&cfg.CloneInfo.Commit, "git-commit", "", "the git commit to use, if it's empty will use the git-ref only")
	pflag.StringVar(&cfg.CloneInfo.ImageName, "image-name", "", "the name of the image to build")
	pflag.StringVar(&cfg.CloneInfo.ImageTag, "image-tag", "", "the tag for the new image")
	pflag.BoolVar(&cfg.CloneInfo.Overwrite, "overwrite", false, "overwrite will remove the directory before cloning")
	pflag.BoolVar(&showVersion, "version", false, "show version")

	pflag.Parse()

	// https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
}

func main() {
	v := version.Get()
	if showVersion {
		b, err := json.Marshal(&v)
		if err != nil {
			fmt.Printf("failed decoding version: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
		os.Exit(0)
	}
	glog.V(4).Infof("Version: %s, GitCommit: %s, GoVersion: %s, BuildDate: %s", v.Version, v.GitCommit, v.GoVersion, v.BuildDate)

	// parse the credentials from an unix environment
	if err := parseCredentials(&cfg); err != nil {
		log.Fatalf("-----> %s", err)
	}

	fmt.Println("-----> Cloning app")
	// Clone the APP
	if err := git.Clone(&cfg); err != nil {
		log.Fatalf("-----> failed cloning app: %v", err)
	}
	// identify the language pack
	buildpack, err := pack.Detect(&cfg)
	if err != nil {
		log.Fatalf("-----> failed detecting language: %v", err)
	}
	if err := buildpack.CreateDockerfile(); err != nil {
		log.Fatalf("-----> failed generating docker file: %v", err)
	}
	fmt.Printf("-----> %s app detected\n", buildpack.CapitalizedPackName())

	fmt.Println("-----> Starting build... but first, cofee!")
	if err := buildpack.RunBuild(); err != nil {
		log.Fatalf("-----> Fail [%v]", err)
	}

	if cfg.PushToRegistry {
		fmt.Println("-----> Pushing to registry")
		if err := buildpack.PushToRegistry(); err != nil {
			log.Fatalf("-----> %s", err)
		}
	}
	fmt.Println("-----> Done!")
}

func parseCredentials(c *conf.Config) error {
	gs, rs := &c.GitSecret, &c.RegistrySecret
	gs.Username, gs.Password = os.Getenv("GIT_USERNAME"), os.Getenv("GIT_PASSWORD")
	gs.OauthToken = os.Getenv("GIT_OAUTH_TOKEN")
	rs.RegUsername, rs.RegPassword = os.Getenv("REGISTRY_USERNAME"), os.Getenv("REGISTRY_PASSWORD")

	if c.GitAuthType() == conf.GitNoAuth {
		if gitBasicAuth, err := ioutil.ReadFile(gitBasicAuthFilePath); err == nil {
			parts := strings.SplitN(string(gitBasicAuth), ",", 2)
			gs.Username = parts[0]
			gs.Password = strings.TrimSpace(parts[1])
		}
		if gitOauthToken, err := ioutil.ReadFile(gitOauthTokenFilePath); err == nil {
			gs.OauthToken = string(gitOauthToken)
		}
	}
	if len(rs.RegUsername) == 0 {
		regData, err := ioutil.ReadFile(registrySecretFilePath)
		if err != nil {
			return fmt.Errorf("failed parsing credentials: %v", err)
		}
		registry := conf.Registry{}
		if err := json.Unmarshal(regData, &registry); err != nil {
			return fmt.Errorf("failed decoding credentials: %v", err)
		}
		regSecret, ok := registry[c.RegistryURL]
		if !ok {
			return fmt.Errorf("failed retrieving registry key: %#v", c.RegistryURL)
		}
		c.RegistrySecret = regSecret
	}
	return nil
}
