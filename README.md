# Factory

Create detectable buildpacks templates with docker images.

Factory allows detecting source code using scripts with Dockerfile templates.

> *Important:* This is an experimental project yet.

## How it Works

The folder `./packs` contains folders containing the language runtimes with two files: 

  - `<runtime>/Dockerfile`
  - `<runtime>/detect`

The `Dockerfile` file could be defined with a GO template entry `{{ .Version }}` which will be populated dynamically by a script.
The `detect` file it's a bash script which contains logic to identify the language and the semantic version of the source code.

When factory starts it executes each `detect` script from the `packs` folder trying to detect the 
source code. When a match is detected (exit code 0), the script must print the semantic version of the detected language which
will be injected into the respective Dockerfile, corresponding to a specific existent docker image version.

```bash
$ fkt --clone-url https://github.com/heroku/node-js-getting-started.git \
    --image-name node-app --registry-auth=eyJ1c2V...DkifQo= --registry-org=acme \
    --registry-url=quay.io --clone-path=/tmp/node-app
-----> Cloning app
Counting objects: 493, done.
Total 493 (delta 0), reused 0 (delta 0), pack-reused 493
-----> Node app detected
-----> Starting build... but first, cofee!
Step 1/4 : FROM node:6.10.2-onbuild
# Executing 5 build triggers...
Step 1/1 : ARG NODE_ENV
 ---> Using cache
Step 1/1 : ENV NODE_ENV $NODE_ENV
 ---> Using cache
Step 1/1 : COPY package.json /usr/src/app/
 ---> Using cache
Step 1/1 : RUN npm install && npm cache clean
 ---> Using cache
Step 1/1 : COPY . /usr/src/app
 ---> 967830d8bcc7
Step 2/4 : EXPOSE 8080
 ---> Running in 6f79f07d2a05
 ---> e205511b037e
Step 3/4 : RUN npm install
 ---> Running in c894619962ad
npm info it worked if it ends with ok
npm info using npm@3.10.10
npm info using node@v6.10.2
npm info lifecycle node-js-getting-started@0.2.6~preinstall: node-js-getting-started@0.2.6
npm info linkStuff node-js-getting-started@0.2.6
npm info lifecycle node-js-getting-started@0.2.6~install: node-js-getting-started@0.2.6
npm info lifecycle node-js-getting-started@0.2.6~postinstall: node-js-getting-started@0.2.6
npm info lifecycle node-js-getting-started@0.2.6~prepublish: node-js-getting-started@0.2.6
npm info ok
 ---> 982d60fa8bd3
Step 4/4 : CMD npm start
 ---> Running in ba1fb2e94da6
 ---> acce5743afb1
Successfully built acce5743afb1
Successfully tagged quay.io/acme/node-app:v1
-----> Pushing to registry
The push refers to a repository [quay.io/acme/node-app]
ef0e37abe726: Pushed
53a0fd72a1c6: Pushed
7f70addbe489: Pushed
806a992360de: Pushed
2af500430b26: Pushed
e71eccb6eee4: Pushed
246ae56dbdbd: Pushed
eda5b29538df: Pushed
d359ab38b013: Pushed
-----> Done
```
