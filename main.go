package main

import "github.com/nokia-paco-automation/paco-parser/cmd"

//go:generate bash -c "/home/hans/git/ygot/generator/generator -output_file=srlygot/SRL_ygot.go -package_name=srlygot -generate_fakeroot -fakeroot_name=device -path=/home/hans/srlinux/21.3.1/yang/yang-adjusted/modules/models/ -typedef_enum_with_defmod  -shorten_enum_leaf_names /home/hans/srlinux/21.3.1/yang/yang-adjusted/modules/models/srl_nokia/models/*/*.yang"

func main() {
	cmd.Execute()
}
