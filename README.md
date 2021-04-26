go run *.go -c conf/paco-deployment.yaml -o out/ parse

go run *.go -c conf/paco-deployment-telenet.yaml -o out/ parse



gcloud container hub memberships register anthos-admin-cluster \
            --context=anthos-admin-cluster-admin@anthos-admin-cluster \
            --service-account-key-file=/home/nokia/baremetal/bmctl-workspace/.sa-keys/anthos-bm-nokia-anthos-baremetal-connect.json \
            --kubeconfig=/home/nokia/baremetal/bmctl-workspace/anthos-admin-cluster/anthos-admin-cluster-kubeconfig \
            --project=anthos-bm-nokia