# How to optimize cost with AWS Graviton and Spot in Amazon Elastic Kubernetes Service (EKS)

## AWS Graviton

Just a quick reminder about AWS Graviton.

AWS Graviton Processors are custom built by Amazon Web Services using 64-bit Arm Neoverse cores to deliver the best price performance for your cloud workloads running in Amazon EC2. AWS Graviton2-based deliver up to 40% better price-performance over comparable current generation x86-based instances1 for a broad spectrum of workloads such as application servers, microservices, video encoding, high-performance computing, electronic design automation, compression, gaming, open-source databases, in-memory caches, and CPU-based machine learning inference.

## Amazon EC2 Spot

Amazon EC2 Spot Instances let you take advantage of unused EC2 capacity in the AWS cloud. Spot Instances are available at up to a 90% discount compared to On-Demand prices. 

## Goals
The goal of this article is to show how to set up a multiarch EKS cluster and use spot managed group.
Indeed diversification across multiple instance types are important with spot so why not adding Graviton2 based instances?
The application is a hello-world web application developed in Java, in go and nodeJS.

So the following step will be done:
* Install all the needed tool in an instance
* Create an EKS cluster
* Create an ECR repository
* Add a spot ARM managed node group
* Create multiarch docker images
   * for a Java application
   * for a go application
   * for a nodeJS application
* Install a deployment for all the created images into EKS
* Verify pods works well in amd64 and arm64 arch.
* Next Step

## Install tools

I use a Linux 2 AMI for this article so some commands can change if you use a different distribution or your laptop.

## Install git, docker, java 15, go, eksctl, kubectl, docker-buildx

```
sudo yum update -y

# install AWS CLI (already installed in Amazon Linux 2)
# https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html


# install git
# https://github.com/git-guides/install-git
sudo yum install git -y

# install docker
# https://docs.docker.com/get-docker/
sudo amazon-linux-extras install docker -y
sudo service docker start
sudo usermod -a -G docker $USER
# You must logout and login or restart to let the system run group policy again and add the current user to the docker group

# verify if everything works
docker info

# This package does not contain the buildx command. So I need to install it manually.
# https://github.com/docker/buildx#installing

# Buildx is still experimental
export DOCKER_CLI_EXPERIMENTAL=enabled

mkdir -p ~/.docker/cli-plugins/
wget -O ~/.docker/cli-plugins/docker-buildx https://github.com/docker/buildx/releases/download/v0.5.1/buildx-v0.5.1.linux-amd64 

chmod a+x ~/.docker/cli-plugins/docker-buildx

# Install Kubectl
# https://docs.aws.amazon.com/eks/latest/userguide/install-kubectl.html
curl -o kubectl https://amazon-eks.s3.us-west-2.amazonaws.com/1.18.9/2020-11-02/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin

kubectl version

# Install eksctl
# https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html

curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
sudo mv /tmp/eksctl /usr/local/bin
eksctl version

```

Let's create a env variable for the region and the account

```
export REGION=eu-west-1
export ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
```


## Create a repository in Amazon Elastic Container Registry (ECR)

To retrieve a password to authenticate to my registry
```
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com
```

Create a repository to push all my hello word application. I will put different hello word implementation done in different language into the same repository but different tags.
```
aws ecr create-repository \
    --repository-name hello-world \
    --image-scanning-configuration scanOnPush=true \
    --region $REGION
```

This image can be used to install emulators for architectures your node does not have native support so that you can run and build containers for any architecture.
```
docker run --privileged --rm tonistiigi/binfmt --install all
```
I can create new instances using the docker buildx create command. This creates a new builder instance with a single node based on my current configuration. 
```
docker buildx create --name builder --use
docker buildx inspect --bootstrap
```

You should see at least in Platforms: linux/amd64 and linux/arm64.

## Create the EKS cluster

Create an EKS cluster with an on-demand x86 managed node group.
```
eksctl create cluster --name=eks-arch-managed-node-groups --instance-types=m5.xlarge,m5a.xlarge,m5d.xlarge --managed --nodes-max=5 --nodes-min=1 --nodes=1 --asg-access --nodegroup-name on-demand-amd-4vcpu-16gb --region=$REGION
```

Add a spot ARM managed node group
```
eksctl create nodegroup --cluster eks-arch-managed-node-groups --instance-types m6gd.xlarge,m6g.xlarge --managed --spot --name spot-4vcpu-16gb --asg-access --nodes-max=5 --nodes-min=1 --nodes=1 --region=$REGION
```

## Docker and mutliarch image

Building multi-architecture Docker images is still an experimental feature. However, hosting multi-architecture images is already well supported by Docker’s Registry. 

```
docker buildx imagetools inspect ubuntu:21.04
```
The result is:
```
Name:      docker.io/library/ubuntu:21.04
MediaType: application/vnd.docker.distribution.manifest.list.v2+json
Digest:    sha256:b6dc45a852dc83fa0e7504e9d68b9b0084eefb8aeb5f295f276bf99f5c033490

Manifests:
  Name:      docker.io/library/ubuntu:21.04@sha256:eb9086d472747453ad2d5cfa10f80986d9b0afb9ae9c4256fe2887b029566d06
  MediaType: application/vnd.docker.distribution.manifest.v2+json
  Platform:  linux/amd64

  Name:      docker.io/library/ubuntu:21.04@sha256:017b74c5d97855021c7bde7e0d5ecd31bd78cad301dc7c701bb99ae2ea903857
  MediaType: application/vnd.docker.distribution.manifest.v2+json
  Platform:  linux/arm/v7

  Name:      docker.io/library/ubuntu:21.04@sha256:bb48336f1dd075aa11f9e819fbaa642208d7d92b7ebe38cb202b0187e1df8ed4
  MediaType: application/vnd.docker.distribution.manifest.v2+json
  Platform:  linux/arm64/v8

  Name:      docker.io/library/ubuntu:21.04@sha256:29c2f09290253a0883690761f411cbe5195cd65a4f23ff40bf66d7586d72ebb7
  MediaType: application/vnd.docker.distribution.manifest.v2+json
  Platform:  linux/ppc64le

  Name:      docker.io/library/ubuntu:21.04@sha256:e8e0c3580fc5948141d8f60c062e0640d4c7e02d10877a19a433573555eda25b
  MediaType: application/vnd.docker.distribution.manifest.v2+json
  Platform:  linux/s390x
  ```

We can see that the image is built for different architectures.
For more information:
https://docs.docker.com/registry/spec/manifest-v2-2/

Docker manages to choose the right version they should pull and run according to the architecture of the docker engine.
So you can use the same command `docker run -t -i --rm ubuntu bash` in amd64 or arm64.

Let's do a hello world web app and push multi architecture docker images into ECR.

I've taken Java for the compiled into bytecode language, for the interpreted language I've chosen nodejs and for the compiled language go.
So with these examples, we have all types of language. For example Python, should use the same way than nodeJS because it's also an interpreted language.

## JAVA

For Java, the code is compiled to Java bytecode and it's architecture-independent. So when you build in arm or amd64 the Java Bytecode will be the same. In this example, I will build the package into docker using a multistage build to avoid installing Java and maven locally. I use `FROM --platform=$BUILDPLATFORM maven:3-openjdk-15-slim AS build` to avoid building the image two times. Indeed the second times the build layer will be cached.
```
cd java
```
### Create the docker image for amd64 and arm64 architecture and push it to ECR
```
docker buildx build --progress plain --platform linux/amd64,linux/arm64 -t $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:java --push .
```
### Verify if the images are available in ECR
```
docker buildx imagetools inspect $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:java
```

I've got another version of a Dockerfile without the multistage part. I need to compile my application into Java bytecodes before building the docker image with the `DockerfileNoStage` file. So let's install a JDK first.

```
# Install a JDK 15
# https://docs.aws.amazon.com/corretto/latest/corretto-15-ug/what-is-corretto-15.html
# or 
# https://adoptopenjdk.net/installation.html
sudo rpm --import https://yum.corretto.aws/corretto.key
sudo curl -L -o /etc/yum.repos.d/corretto.repo https://yum.corretto.aws/corretto.repo
sudo yum install -y java-15-amazon-corretto-devel

java --version
```

### compile and package the app with the maven wrapper tool
```
./mvnw clean package
```
### Create the docker image for amd64 and arm64 architecture and push it to ECR
```
docker buildx build --progress plain --platform linux/amd64,linux/arm64 -t $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:javaNoStage -f DockerfileNoStage --push .
```
### Verify if the images are available in ECR
```
docker buildx imagetools inspect $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:javaNoStage
```
## Golang
```
cd ../golang
```
Go need to be built for different architectures. I use a multistage dockerfile, to get the right building image during the build process.

### Create the docker image for amd64 and arm64 architecture and push it to ECR
```
docker buildx build --progress plain --platform linux/amd64,linux/arm64 -t $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:go --push .
```
### Verify if the images are available in ECR
```
docker buildx imagetools inspect $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:go
```
Go is also able to build apllication for different Operating Systems and architectures by using environment variables. So I have another Dockerfile version witch used the application already built locally.
```
# Install golang
sudo yum install golang -y
# Create the go application for arm64 architecture and name it 'arm64'.
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -o arm64 .
# Create the go application for amd64 architecture and name it 'amd64'.
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o amd64 .
```

There is another version of the Dockerfile for the go application `DockerfileBuildSamePlatform`. This version avoid pulling another version of the build image.

## NodeJS

```
cd ../nodejs
```
For nodeJS the application doesn't to be build or package.

### Create the docker image for amd64 and arm64 architecture and push it to ECR
```
docker buildx build --progress plain --platform linux/amd64,linux/arm64 -t $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:nodejs --push .
```
### Verify if the images are available in ECR
```
docker buildx imagetools inspect $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/hello-world:nodejs
```

## Deploy the applications into EKS

Install gettext to use ensubst to replace env variable in the yaml deployment file.
```
yum install gettext
```

```
envsubst < go.yaml | kubectl apply -f -

envsubst < java.yaml | kubectl apply -f -

envsubst < javaNostage.yaml | kubectl apply -f -

envsubst < nodejs.yaml | kubectl apply -f -
```

View the pods
```
kubectl get pod -o wide
```
The Java, go, nodejs application work well in amd64 and arm64.

To try a specific application and pod you can use the current ip of a pod inside a container with the image curlimages/curl:
```
kubectl run -i --tty curlimage --image=curlimages/curl -- sh

curl http://${ip-of-a-pod}:${app-port}
```

### Delete the cluster
```
eksctl delete cluster eks-arch-managed-node-groups

aws ecr delete-repository \
    --repository-name hello-world \
    --region $REGION
````

## Conclusion
In this demo, we create multi-arch docker images and use them with EKS. 
By combining Graviton 2 and spot, the potential saving is very important. For this demo, I used 2 different managed node group for ARM and x86 to be sure to deploy my pod into the 2 architectures but the different architecture could be into the same managed node group.
