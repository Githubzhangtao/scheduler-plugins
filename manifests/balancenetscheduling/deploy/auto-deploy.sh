# 实验1
#!/bin/sh
#declare i=0
#for var2 in {1..5}
#do
#  for var1 in 10 20 10 20 40
#  do
#    let i++
#    echo "apiVersion: v1
#kind: Pod
#metadata:
#  name: nginx-$var1-$i
#  labels:
#    netRequest: '$var1'
#spec:
#  containers:
#    - name: nginx
#      imagePullPolicy: Never
#      image: nginx:latest
#
#    " > deploy-nginx-pod.yaml
#		kubectl apply -f deploy-nginx-pod.yaml
#		sleep 1s
#  done
#done

# 实验2
#!/bin/sh
declare i=0
for c in 200 300 1500
do
  for m in 100 400 2000
  do
    for n in 3 5 20
    do
      let i++
      echo "apiVersion: v1
kind: Pod
metadata:
  name: $i-${c}m-${m}mi-${n}mb
  labels:
    netRequest: '$n'
spec:
  containers:
    - name: nginx
      imagePullPolicy: Never
      image: nginx:latest
      resources:
        limits:
            cpu: ${c}m
            memory: ${m}Mi
        requests:
            cpu: ${c}m
            memory: ${m}Mi
    " > deploy-nginx-pod.yaml
#    cat  deploy-nginx-pod.yaml
		kubectl apply -f deploy-nginx-pod.yaml
		sleep 0.5s
    done
  done
done

