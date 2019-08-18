#!/bin/bash

if [ "$1" == "" ] 
then
    exit 1
fi
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

export KUBECONFIG=$DIR/kubeconfig.yaml
cat $DIR/cronjob.yaml |sed "s/VERSION/$1/" | kubectl apply -f -

kubectl delete secret gofish.github
kubectl create secret generic gofish.github --from-env-file=$DIR/../.env