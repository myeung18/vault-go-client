
*** A Go library for KMS (vault and CyberArk)


#### Vault setup on OCP

```shell

# 
helm install vault hashicorp/vault \
--set "global.openshift=true" \
--set "server.dev.enabled=true"

# inside pod
oc exec -it vault-0 -- /bin/sh

vault auth enable kubernetes

vault write auth/kubernetes/config \
    kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443"

# outside pod
oc apply --filename service-account-webapp.yml

# inside pod
oc exec -it vault-0 -- /bin/sh

# create secret in Vault
vault kv put secret/webapp/config username="static-user" \
    password="static-password"

vault kv get secret/webapp/config

# create read policy
vault policy write webapp - <<EOF
path "secret/data/webapp/config" {
  capabilities = ["read"]
}
EOF

# update policy for server account in certain namespace
$ vault write auth/kubernetes/role/webapp \
    bound_service_account_names=webapp \
    bound_service_account_namespaces=vault \
    policies=webapp \
    ttl=24h
    
# outside pod
$ oc apply --filename deployment-webapp.yml

# run the app
oc exec \
   $(oc get pod --selector='app=webapp' --output='jsonpath={.items[0].metadata.name}') \
   --container app -- curl -s http://localhost:8080 ; echo

```

#### Conjur app deployment

```shell

# deploy and configure conjur-oss

oc apply -f deployment-conjur.yml -n data-team1
```
