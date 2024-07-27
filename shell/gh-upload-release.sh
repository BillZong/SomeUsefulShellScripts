#!/bin/bash

SYSNAME=`uname`

# params analyzing
if [ ${SYSNAME}='Darwin' ]; then
    # this shell only works on Mac and bash now.
    ARGS=`getopt ht:k:c:drf:p: $@`
elif [ ${SYSNAME}='Linux' || ${SYSNAME}='Unix' ]; then
    # this only works on linux/unix.
    ARGS=`getopt -o ht:k:c:drf:p: -l help,tag:,token:,target-commitish:,draft,prerelease,archive-files:directory-path: -- "$@"`
else
    echo "Windows not supported yet"
fi

if [ $? != 0 ]; then
    echo "Terminating..."
    exit 1
fi

eval set -- "${ARGS}"

function showHelp {
	echo "Upload current repo (latest) tag info and files to Github release page"
	echo ""
	echo "Usage: $0 [options]"
	echo "	-t|--tag tag (default: current repo's latest one)"
	echo "	-k|--token token"
	echo "	-c|--target-commitish target_commitish (default: 'master')"
	echo "	-d|--draft (default: disable, means false)"
	echo "	-r|--prerelease (default: disable, means false)"
	echo "	-f|--archive-files files (seperated by ',')"
    echo "  -p|--directory-path directory_path"
}

while true
do
    case "${1}" in
        -h|--help)
            shift
            showHelp
            exit 0
            ;;
        -t|--tag)
            tag="$2"
            shift 2
            ;;
        -k|--token)
            token="$2"
            shift 2
            ;;
        -c|--target-commitish)
            target="$2"
            shift 2
            ;;
        -d|--draft)
            isDraft="true"
            shift
            ;;
        -r|--prerelease)
            isPrerelease="true"
            shift
            ;;
        -f|--archive-files)
            fileArray=(${2//,/ })
            shift 2
            ;;
        -p|--directory-path)
            directoryPath="$2"
            shift 2
            ;;
        --)
            shift;
            break;
            ;;
        *)
            echo "Could not support that params $1"
            exit 1
            ;;
    esac
done

origin=`git remote -v | grep origin | awk 'NR==1 {print $2}'`
if [ -z "$origin" ]; then
    echo "not in a git repo"
    exit 1
fi

if [ -z "$userID" ]; then
	userID=`echo $origin | awk -F "/" '{print $4}'`
fi
if [ -z "$repoName" ]; then
    repoName=`echo $origin | awk -F "/" '{print $5}'`
    repoName=${repoName%.*}
fi
if [ -z "$tag" ]; then
	tag=`git tag -l | tail -n 1`
fi
if [ -z "$token" ]; then
	echo "must provide token"
	exit 1
fi
if [ -z "$target" ]; then
	target="master"
fi

if [ -z `which github-release` ]; then
    go get github.com/aktau/github-release
fi

# read tag info
tagInfo=`git cat-file -p $(git rev-parse $tag) | tail -n +6`
if [ -z "$tagInfo" ]; then
	tagInfo="No tag info"
fi

# set your token
export GITHUB_TOKEN=$token

# create release
cmd="github-release release \
--user $userID \
--repo $repoName \
--tag $tag \
--name $tag \
--description \"$tagInfo\""

if [ -n "$isDraft" ]; then
    cmd="$cmd --draft"
fi
if [ -n "$isPrerelease" ]; then
    cmd="$cmd --pre-release"
fi

eval $cmd

# upload files
for f in ${fileArray[@]}; do
    github-release upload -u $userID -r $repoName -t $tag -f $f -n ${f#*/}
done

# upload all files in directory, depth 1
if [ -n "$directoryPath" ]; then
    ls $directoryPath/* | while read f; do echo "github-release upload -u $userID -r $repoName -t $tag -f $f -n ${f#*/}";github-release upload -u $userID -r $repoName -t $tag -f $f -n ${f#*/}; done
fi
