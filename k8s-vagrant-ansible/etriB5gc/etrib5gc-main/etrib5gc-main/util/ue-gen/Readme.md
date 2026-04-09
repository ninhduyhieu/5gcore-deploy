A simple program to generate Ue profile and save it to UDR

## Build
From {project-root} run:

```bash
make ue-gen
```

built binary locates at {project-root}/bin

## Show helps

```bash
./bin/ue-gen

NAME:
   uegen - Ue subscription data and profile generator

USAGE:
    [global options] command [command options] [arguments...]

COMMANDS:
   init     init/reset UDR data
   gen      generate UE yaml profiles and subscription data
   save     Save UE to UDR
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```

## Operator configuration

All commands require a configuration for the network.

```json
{
	"udr": {
		"url": "mongodb://127.0.0.1:27017",
	  	"dbName": "etrib5gc",
	  	"authSub": "authsub",
	  	"amSub": "amsub",
	  	"smfSel": "smfSel",
	  	"smSub": "smSub"
	},
	"plmnId" : {
		"mcc": "208",
		"mnc":"93"
	},
	"profiles":[
		{
			"scheme": 1,
			"prvkey":       "c53c22208b61860b06c62e5406a7b330c2b577aa5558981510d128247d38bd1d",
			"pubkey":        "5a8d38864820197c3394b92613b20b91633cbd897119273bf8e4a6f4eec0a650"
		},
		{
			"scheme": 2,
			"prvkey":       "F1AB1074477EBCC7F554EA1C5FC368B1616730155E0041AC447D6301975FECDA",
			"pubkey":        "0272DA71976234CE833A6907425867B82E074D44EF907DFB4B3E21C1C2256EBCD1"
		}
	],
	"amf": "8000",
	"slices": [{
		"id" : {
			"sst" : 1,
			"sd": "0x010203"
		},
		"dnnList": ["internet"]
	},{
		"id": {
			"sst" : 1,
			"sd": "0x54321"
		},
		"dnnList": ["internet"]
	}]
}
```
Most information should be taken as it is from the NSM configuration.

## Generate UEs
		
```bash
./ue-gen gen --help

NAME:
    gen - generate UE yaml profiles and subscription data

USAGE:
    gen [command options] [arguments...]

OPTIONS:
   --config FILE, -c FILE    Load configuration from FILE
   --prefix value, -p value  Prefix for naming UE profile (default: "ue_")
   --numUes value, -n value  Number of UEs to generate (default: 1)
   --outputDir DIR, -d DIR   output directory for UE profiles DIR (default: ".")
   --inc, -i                 Create UEs with incremental SUPI value
   --imsi value              Starting imsi value (default: 1)
   

```

### Random UEs

To generate 10 UEs, save Ue profiles in folder `uelist` and name them with prefix `ue-` and add their subscription data into the UDR:

```bash
mkdir uelist
./bin/ue-gen gen -c uegen.json -n 10 -d uelist -p "ue-"
```

The above command will create random UEs with default paramters. You can
generate UEs from a template. Just copy an existing UE profile yaml file, name
it as "ue.yaml" then put it under the output folder. The new UEs will be
generate based on this template. They are sharing most attributes except for
`supi`, suci protection related attributes, imei.  To set the protection
scheme, set the value of `homeNetworkPuclicKeyId` to the index (starting from 1) 
of a profile in the operator's configuration, the corresponding scheme of the
chosen profile will be set for the UEs. If the value is set to any number that
is out of range of the profiles, the Null protection scheme will be selected
for the UEs.

### UEs with incremental SUPIs
For conducting experiments with large number of UEs, it is convinient if we can generate UEs that can share a single profile with predictable SUPI. 

```bash
./bin/ue-gen gen -c uegen.json -d uelist -n 1000 -i --imsi 1
```

The command will generate 1000 UEs sharing all of subscription data except for the SUPI values. The output will be  a single yaml file with SUPI whose embeded IMSI value is 1. When conducting experiment, you should load this profile and change the SUPI incrementally for the other UEs.


## Save UE subscription data

When you need to add subscription data for COTSUE, prepare a yaml profile file with the information from the COTSUE then run this command to save the subscription data to the UDR.
Make sure that the PlmnId from the yaml file and from the operator configuration file are the same (the program will not check their equality)

```bash
./ue-gen save --help

NAME:
    save - Save UE to UDR

USAGE:
    save [command options] [arguments...]

OPTIONS:
   --config FILE, -c FILE  Load configuration from FILE
   --ue FILE, -p FILE      UE yaml profile FILE
```
