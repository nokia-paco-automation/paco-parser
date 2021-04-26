# Paco parser

generates the values.yml files for the Nokia Packet core from the IaaC Yaml files

## Example

Multi-netting example:

```
go run *.go -c conf/paco-deployment-telenet-multinet.yaml -o out/ parse
```

VlanAwareApp example:

```
go run *.go -c conf/paco-deployment-telenet-vlanawareapp.yaml -o out/ parse
```