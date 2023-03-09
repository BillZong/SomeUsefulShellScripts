if [ $# -lt 1 ];then
  echo "Usage: $0 object-to-remove-from-current-repo"
  exit -1
fi

git filter-branch --force --index-filter "git rm --cached -f --ignore-unmatch $1" --prune-empty --tag-name-filter cat -- --all
