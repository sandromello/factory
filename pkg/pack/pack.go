package pack

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/golang/glog"

	"github.com/sandromello/factory/pkg/conf"
	"github.com/sandromello/factory/pkg/pack/generated"
)

type Buildpack struct {
	Name       string
	Output     []byte
	Dockerfile []byte

	client *docker.Client
	cfg    *conf.Config
}

func Detect(cfg *conf.Config) (bp *Buildpack, err error) {
	bp = &Buildpack{}
	for _, packName := range []string{"python", "php", "node"} {
		detectScript, err := generated.Asset(fmt.Sprintf("%s/detect", packName))
		if err != nil {
			return nil, fmt.Errorf("failed getting detect script: %v", err)
		}
		cmd := exec.Command("/bin/bash", "-s", cfg.CloneInfo.Path)
		cmd.Stdin = bytes.NewBuffer(detectScript)
		bp.Output, err = cmd.Output()
		if err != nil {
			glog.V(4).Infof("%s - failed detecting pack [%#v]", packName, err.Error())
			continue
		}
		bp.Dockerfile, err = generated.Asset(fmt.Sprintf("%s/Dockerfile", packName))
		if err != nil {
			return nil, err
		}
		bp.Name = packName
		bp.cfg = cfg
		glog.V(4).Infof("%s - found pack!", packName)
		break
	}
	return
}

func (b *Buildpack) CapitalizedPackName() string {
	return strings.Title(b.Name)
}

func (b *Buildpack) CreateDockerfile() error {
	t, err := template.New("Dockerfile").Parse(string(b.Dockerfile))
	if err != nil {
		return fmt.Errorf("failed parsing template: %v", err)
	}
	f, err := os.Create(filepath.Join(b.cfg.CloneInfo.Path, "Dockerfile"))
	if err != nil {
		glog.V(4).Infof("%s - failed creating Dockerfile [%v]", b.Name, err)
		return err
	}
	defer f.Close()
	return t.Execute(f, struct{ Version string }{
		Version: strings.TrimSpace(string(b.Output)),
	})
}

func (b *Buildpack) RunBuild() error {
	d, err := docker.NewClient(b.cfg.DockerAddr, "", nil, nil)
	if err != nil {
		return fmt.Errorf("Failed connecting to docker socket: %v", err)
	}
	b.client = d

	buildContext, err := createTarStream(b.cfg.CloneInfo.Path)
	if err != nil {
		return fmt.Errorf("Failed creating tarball: %v", err)
	}
	defer buildContext.Close()

	imageTag := b.cfg.GetImageTag()
	imagePrefix := b.cfg.RegistryOrg
	if imagePrefix != "" {
		imagePrefix = b.cfg.RegistryOrg + "/"
	}
	imageName := fmt.Sprintf("%s/%s%s:%s", b.cfg.RegistryURL, imagePrefix, b.cfg.CloneInfo.ImageName, imageTag)
	glog.V(4).Infof("%s - building image %s", b.Name, imageName)
	buildResp, err := d.ImageBuild(context.Background(), buildContext, types.ImageBuildOptions{
		Tags: []string{imageName},
	})
	if err != nil {
		return fmt.Errorf("Failed starting build: %v", err)
	}

	defer buildResp.Body.Close()
	if err := jsonmessage.DisplayJSONMessagesStream(buildResp.Body, os.Stderr, os.Stderr.Fd(), true, nil); err != nil {
		return fmt.Errorf("failed streaming image build response: %v", err)
	}
	return nil
}

func (b *Buildpack) PushToRegistry() error {
	imageTag := b.cfg.GetImageTag()
	imagePrefix := b.cfg.RegistryOrg
	if imagePrefix != "" {
		imagePrefix = b.cfg.RegistryOrg + "/"
	}
	imageName := fmt.Sprintf("%s/%s%s:%s", b.cfg.RegistryURL, imagePrefix, b.cfg.CloneInfo.ImageName, imageTag)
	pushResp, err := b.client.ImagePush(context.Background(), imageName, types.ImagePushOptions{
		RegistryAuth: b.cfg.RegistryAuth(),
	})
	if err != nil {
		return fmt.Errorf("Failed pushing to registry: %v", err)
	}
	defer pushResp.Close()
	if err := jsonmessage.DisplayJSONMessagesStream(pushResp, os.Stderr, os.Stderr.Fd(), true, nil); err != nil {
		return fmt.Errorf("failed streaming image push response: %v", err)
	}
	return nil
}

func createTarStream(path string) (io.ReadCloser, error) {
	tarOpts := &archive.TarOptions{
		ExcludePatterns: []string{},
		IncludeFiles:    []string{"."},
		Compression:     archive.Uncompressed,
		NoLchown:        true,
	}
	return archive.TarWithOptions(path, tarOpts)
}
