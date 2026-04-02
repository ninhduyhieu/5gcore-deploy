package main

import (
	"encoding/json"
	"etrib5gc/logctx"
	"etrib5gc/nfs/udr"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli"
	"math/rand"
	"time"
)

var log *logrus.Entry

// type of opc
const (
	OPC = "OPC"
	OP  = "OP"
)
const (
	DEFAULT_ROUTING_INDICATOR = "0000"
)

var _incremental bool
var _numUes int
var _uePrefix, _dir string
var _imsi int

func main() {
	cfgFlag := cli.StringFlag{
		Name:     "config, c",
		Usage:    "Load configuration from `FILE`",
		Required: true,
	}

	genFlags := []cli.Flag{
		cfgFlag,
		cli.StringFlag{
			Name:        "prefix, p",
			Usage:       "Prefix for naming UE profile",
			Destination: &_uePrefix,
			Value:       "ue_",
		},
		cli.IntFlag{
			Name:        "numUes, n",
			Usage:       "Number of UEs to generate",
			Destination: &_numUes,
			Value:       1,
		},
		cli.StringFlag{
			Name:        "outputDir, d",
			Usage:       "output directory for UE profiles `DIR`",
			Destination: &_dir,
			Value:       ".",
		},
		cli.BoolFlag{
			Name:        "inc, i",
			Usage:       "Create UEs with incremental SUPI value",
			Destination: &_incremental,
		},
		cli.IntFlag{
			Name:        "imsi",
			Usage:       "Starting imsi value",
			Destination: &_imsi,
			Value:       1,
		},
	}

	app := &cli.App{
		Name:  "uegen",
		Usage: "Ue subscription data and profile generator",

		Commands: []cli.Command{
			cli.Command{
				Name:   "init",
				Usage:  "init/reset UDR data",
				Action: initUdr,
				Flags: []cli.Flag{
					cfgFlag,
				},
			},
			cli.Command{
				Name:   "gen",
				Usage:  "generate UE yaml profiles and subscription data",
				Flags:  genFlags,
				Action: generateUe,
			}, cli.Command{
				Name:  "save",
				Usage: "Save UE to UDR",
				Flags: []cli.Flag{
					cfgFlag,
					cli.StringFlag{
						Name:  "ue, p",
						Usage: "UE yaml profile `FILE`",
					},
				},
				Action: saveUe,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Fail to start: %+v\n", err)
		return
	}

}

func generateUe(c *cli.Context) error {
	//read config

	cfg, err := loadConfig(c.String("config"))
	if err != nil {
		return err
	}

	log = logctx.Entry(logrus.Fields{
		"app": "uegen",
	})

	//read ue template file
	_dir = strings.TrimRight(_dir, "/")
	var ueTemplate *UeProfile
	fn := _dir + "/ue.yaml"
	if ueTemplate, err = loadUe(fn); err != nil {
		log.Warnf("Fail to load Ue profile: %+v", err)
		log.Infof("Use a default template to generate UE")
		ueTemplate = &_defaultUe
	}

	o := newOperator(cfg)

	//create database connection
	cfg.Db.Mock = nil //make sure we use Db
	db := udr.New(&cfg.Db)
	if err = db.Connect(false); err != nil {
		return fmt.Errorf("Failed to connect to database: %+v", err)
	}

	defer db.Close() //close database

	rand.Seed(time.Now().UnixNano())

	ueZero := o.copyTemplate(ueTemplate, _incremental)
	supiGen := o.randSupi
	if _incremental {
		supiGen = o.makeIncSupiGen(_imsi)
	}
	plmnId := &o.config.PlmnId
	for i := range _numUes {
		ue := o.generateUe(ueZero, _incremental, supiGen)
		if !_incremental || i == 0 {
			ueFile := _dir + "/" + _uePrefix + strconv.Itoa(i+1) + ".yaml"
			if err = ue.saveYamlProfile(ueFile); err != nil {
				log.Errorf("save ue profile %d to yaml file failed: %+v", i+1, err)
				return nil
			}
		}
		if err = ue.saveDb(plmnId, db); err != nil {
			log.Errorf("save ue %d to database failed: %+v", i+1, err)
			return nil
		}
	}

	return nil
}

func saveUe(c *cli.Context) error {

	cfg, err := loadConfig(c.String("config"))
	if err != nil {
		return err
	}
	var ue *UeProfile
	if ue, err = loadUe(c.String("ue")); err != nil {
		return err
	}

	log = logctx.Entry(logrus.Fields{
		"app": "uegen",
	})

	cfg.Db.Mock = nil //make sure we use Db
	db := udr.New(&cfg.Db)
	if err = db.Connect(false); err != nil {
		return fmt.Errorf("Failed to connect to database: %+v", err)
	}

	defer db.Close() //close database
	if err = ue.saveDb(&cfg.PlmnId, db); err != nil {
		log.Errorf("save ue to database failed: %+v", err)
	} else {
		log.Infof("Ue subscription data is saved to UDR")
	}

	return nil
}

func initUdr(c *cli.Context) error {
	cfg, err := loadConfig(c.String("config"))
	if err != nil {
		return err
	}
	log = logctx.Entry(logrus.Fields{
		"app": "uegen",
	})

	db := udr.New(&cfg.Db)
	defer db.Close()                        //close database
	if err = db.Connect(true); err != nil { //Drop then create collections again
		log.Errorf("Fail to init UDR: %+v", err)
	}
	return nil
}

func loadConfig(fn string) (*OperatorConfig, error) {
	var cfg OperatorConfig
	buf, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, fmt.Errorf("Fail to read config file: %+v", err)
	} else {
		if err = json.Unmarshal(buf, &cfg); err != nil {
			return nil, fmt.Errorf("Fail to decode json config file: %+v", err)
		}
	}
	return &cfg, nil
}

func loadUe(fn string) (*UeProfile, error) {

	var ue UeProfile
	buf, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, fmt.Errorf("Fail to read ue profile file `%s`: %+v", fn, err)
	} else {
		if err = yaml.Unmarshal(buf, &ue); err != nil {
			return nil, fmt.Errorf("Fail to decode yaml ue template file `%s`: %+v", fn, err)
		} else {
			ue.rewriteSlices()
		}
	}
	return &ue, nil
}
