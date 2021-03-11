#!/bin/sh
var2=(20 30 50)
for var1 in {1..18}
do
  echo $((var2[$(((var1-1) % 3))]))
  netRequest=$((var2[$(((var1-1) % 3))]))
  echo "apiVersion: v1
kind: Pod
metadata:
  name: nginx-${var1}-$netRequest
  labels:
    netRequest: '$netRequest'
spec:
  containers:
    - name: nginx
      image: nginx
      resources:
        limits:
            cpu: 100m
            memory: 100Mi
        requests:
            cpu: 100m
            memory: 100Mi
    " > deploy-nginx-pod.yaml
		kubectl apply -f deploy-nginx-pod.yaml
		sleep 1s
	done
done


##!/bin/sh
#for var1 in 1 2 3
#do
#	for var2 in 100 200 300
#	do
#		echo "apiVersion: v1
#kind: Pod
#metadata:
#  name: nginx-${var1}-$var2
#  labels:
#    netRequest: '$var2'
#spec:
#  containers:
#    - name: nginx
#      image: nginx
#      resources:
#        limits:
#            cpu: 100m
#            memory: 100Mi
#        requests:
#            cpu: 100m
#            memory: 100Mi
#    " > deploy-nginx-pod.yaml
#		kubectl apply -f deploy-nginx-pod.yaml
#		sleep 1s
#	done
#done
