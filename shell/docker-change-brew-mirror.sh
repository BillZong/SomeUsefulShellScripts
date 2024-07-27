#!/bin/bash

TARGET=$1
if [ l"${TARGET}" != l"ustc" -a \
    l"${TARGET}" != l"tsinghua" -a \
    l"${TARGET}" != l"aliyun" -a \
    l"${TARGET}" != l"github" -a; then
    echo "please specify target:"
    echo "    ustc for 中科大镜像"
    echo "    tsinghua for 清华镜像"
    echo "    aliyun for 阿里云镜像"
    echo "    github for 官方仓库"
    exit 255
fi

echo "Well, these docker images are all deprecated, please don't use them anymore."

case ${TARGET} in
    ustc )
        BREW_REPO="https://mirrors.ustc.edu.cn/brew.git"
        BREW_CORE_REPO="https://mirrors.ustc.edu.cn/homebrew-core.git"
        ;;
    tsinghua )
        BREW_REPO="https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/brew.git"
        BREW_CORE_REPO="https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/homebrew-core.git"
        ;;
    aliyun )
        BREW_REPO=""
        BREW_CORE_REPO=""
        ;;
    github )
        BREW_REPO=""
        BREW_CORE_REPO=""
        ;;
esac

SCRIPT_DIR=$(dirname $0)
pushd $SCRIPT_DIR > /dev/null

REGISTRY="registry.cn-shenzhen.aliyuncs.com"
REPOSITORY="fdn2"

# commit sha1 hash
COMMIT_HASH=$(git rev-parse HEAD)

# first 8 characters of COMMIT_HASH
IMAGE_TAG=${COMMIT_HASH:0:8}

IMAGE_URL="${REGISTRY}/${REPOSITORY}/${IMAGE_NAME}:${IMAGE_TAG}"

echo "request:"
echo "    image: ${IMAGE_URL}"
echo "    dockerfile: ${DOCKERFILE_NAME}"
echo ""

# use current directory Dockerfile to build image
echo docker build -f ./dockerfiles/${DOCKERFILE_NAME} -t ${IMAGE_URL} .
docker build -f ./dockerfiles/${DOCKERFILE_NAME} -t ${IMAGE_URL} .
if [ $? -ne 0 ]; then
    echo "build failed!"
    popd
    exit 255
fi

echo docker push ${IMAGE_URL}
docker push ${IMAGE_URL}
if [ $? -ne 0 ]; then
    echo "push failed!"
    popd
    exit 255
fi

echo "image ${IMAGE_URL} pushd!"
echo "${IMAGE_URL}"

popd > /dev/null
