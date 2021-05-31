#!/bin/bash
# 注意请在国外的服务器上执行此脚本
# 先通过下面的命令来拉取image列表
# kubeadm config --kubernetes-version 1.13.3 images list
# 将image列表写入下面的array中
# images=(k8s.gcr.io/kube-apiserver:v1.13.3 k8s.gcr.io/kube-controller-manager:v1.13.3 k8s.gcr.io/k
# 注意填写自己的docker hub的地址

# 使用版本参数
VERSION=${1:-1.19.3}
SEM_VERSION=${VERSION##*v}
echo "$VERSION"
echo "$SEM_VERSION"

images=`kubeadm config --kubernetes-version $SEM_VERSION images list`

mydockerhubprefix=registry.qzcloud.com/gcr
for origin_image in ${images[*]}
do
	my_image=${origin_image/k8s.gcr.io/$mydockerhubprefix}
	echo "VPN服务下载并推送这些镜像，请执行下面的指令："
	echo "docker pull ${origin_image}"
	echo "docker tag ${origin_image} ${my_image}"
	echo "docker push ${my_image}"
done

for origin_image in ${images[*]}
do
	my_image=${origin_image/k8s.gcr.io/$mydockerhubprefix}
	echo "准备手动安装K8S集群的服务器，请执行下面的指令："
	echo "docker pull ${my_image}"
	echo "docker tag ${my_image} ${origin_image}"
done
