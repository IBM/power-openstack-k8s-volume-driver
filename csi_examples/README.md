### IBM PowerVC CSI Driver Sample Scripts

#### Volume Creation

Run the following command to create a volume:

```
oc apply -f <path to csi examples directory>/static-pvc.yaml
```

On your PowerVC you should see a volume created under 'data volumes' section. 

```
'oc get pv/pvc' should list all persistent volume claims. 
```
#### List all Persistent Volumes

```
oc get pv
```

#### List all Persistent Volume Claims

```
oc get pvc
```

#### Delete a Persistent Volume

```
oc delete pv <persistent volume name>
```

#### Delete a Persistent Volume Claim:

```
oc delete pvc <persistent volume claim name>
```

#### Volume Attach

1. Execute this command to create an example pod:
```
oc apply -f <path to csi examples directory>/dynamic-pod.yaml
```

2. After a moment, on PowerVC, you should see the volume getting attached to the worker node.
3. The following command should list the pod in running status:

```
# oc get pods
NAME                                   READY   STATUS             RESTARTS   AGE
example-pod                            1/1     Running            0          2m13s
ibm-powervc-csi-attacher-plugin-0      3/3     Running            0          8m49s
ibm-powervc-csi-plugin-bj4bq           3/3     Running            1          8m48s
ibm-powervc-csi-plugin-mpgwf           3/3     Running            1          8m48s
ibm-powervc-csi-plugin-vd4tr           3/3     Running           1          8m48s
ibm-powervc-csi-provisioner-plugin-0   3/3     Running            0          8m49s
ibm-powervc-csi-resizer-plugin-0       3/3     Running   0 8m49s
```
*Note: "example-pod" is the name used in sample script.*

4. Describe the newly created pod.

5. It should show mypvc is mounted inside the pod as **/usr/share/nginx/html/powervc**.
*(Note: mypvc is the name used in the sample script)*

6. Now, log into the pod and verify the directory:

```
# oc rsh example-pod
# cd /usr/share/nginx/html/
# ls -la
total 12
drwxr-xr-x. 1 root root   21 Feb 24 20:16 .
drwxr-xr-x. 1 root root   18 Feb  2 02:23 ..
-rw-r--r--. 1 root root  494 Jan 21 13:36 50x.html
-rw-r--r--. 1 root root  612 Jan 21 13:36 index.html
drwxr-xr-x. 3 root root 4096 Feb 24 20:15 powervc
# cd powervc
# ls -la
total 20
drwxr-xr-x. 3 root root  4096 Feb 24 20:15 .
drwxr-xr-x. 1 root root    21 Feb 24 20:16 ..
drwx------. 2 root root 16384 Feb 24 20:15 lost+found
#
```

7. Log in to the worker node and check the mount :

```
ssh core@infnod-0.ocp-ppc64le-test-ce90a2.redhat.com
[core@infnod-0 ~]$ mount | grep sdc
/dev/sdc on /var/lib/kubelet/pods/785cfd62-b7de-4c89-9156-a573fb7ca039/volumes/kubernetes.io~csi/pvc-5a58daab-6ad1-48c8-bf87-f377b4b830a6/mount type ext2 (rw,relatime,seclabel,stripe=8)
[core@infnod-0 ~]$
```
