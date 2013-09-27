package agent

import (
	"fmt"
	"os/exec"
	"strings"
	"github.com/garyburd/redigo/redis"
)

type TwemproxyConfig struct {
	Listen       string   `yaml:"listen,omitempty"`
	Hash         string   `yaml:"hash,omitempty"`
	Distribution string   `yaml:"distribution,omitempty"`
	Redis        bool     `yaml:"redis,omitempty"`
	RetryTimeout int      `yaml:"server_retry_timeout,omitempty"`
	FailureLimit int      `yaml:"server_failure_limit,omitempty"`
	Servers      []string `yaml:"servers,omitempty"`
}

var twemproxyConfig map[string]TwemproxyConfig

func UpdateMaster(master_name string, ip string, port string) bool {
	address := ComposeRedisAddress(ip, port)
	Debug(fmt.Sprintf("Updating master %s to %s.", master_name, address))
	servers := twemproxyConfig[Settings.TwemproxyPoolName].Servers
	for i := range servers {
		server_data    := strings.Split(servers[i], string(' '))
		address_data   := strings.Split(server_data[0], string(':'))
		old_address    := ComposeRedisAddress(address_data[0], address_data[1])
		server_name    := server_data[1]

		if master_name == server_name && address != old_address {
			twemproxyConfig[Settings.TwemproxyPoolName].Servers[i] = fmt.Sprint(address, ":1 ", master_name)
			return true
		}
	}

	return false
}

func LoadTwemproxyConfig() {
	Debug("Loading Twemproxy config.")
	ReadYaml(Settings.TwemproxyConfigFile, &twemproxyConfig)
}

func SaveTwemproxyConfig() {
	Debug("Saving Twemproxy config.")
	WriteYaml(Settings.TwemproxyConfigFile, &twemproxyConfig)
}

func RestartTwemproxy() error {
	Debug("Restarting Twemproxy.")
	out, err := exec.Command(Settings.RestartCommand).Output()

	if err != nil {
		Debug(fmt.Sprintf("Cannot restart twemproxy. output: %s. error: %s", out, err))
	}

	return err
}

func GetSentinel() (sentinel string) {
	address := ComposeRedisAddress(Settings.SentinelIp, Settings.SentinelPort)
	return address
}

func SwitchMaster(master_name string, ip string, port string) error {
	Debug("Received switch-master.")
	if UpdateMaster(master_name, ip, port) {
		SaveTwemproxyConfig()
		err := RestartTwemproxy()
		return err
	} else {
		return nil
	}
}

func ValidateCurrentMaster() error {
	c, err := redis.Dial("tcp", GetSentinel())
	if err != nil {
		return err
	}

	reply, err := redis.Values(c.Do("SENTINEL", "masters"))

	if err != nil {
		return err
	}

	var sentinel_info []string

	reply, err = redis.Scan(reply, &sentinel_info)
	if err != nil {
		return err
	}
	master_name := sentinel_info[1]
	ip          := sentinel_info[3]
	port        := sentinel_info[5]

	err = SwitchMaster(master_name, ip, port)

	return err
}

func SubscribeToSentinel() {
	sentinel := GetSentinel()
	c, err := redis.Dial("tcp", sentinel)
	if err != nil {
		Fatal("Cannot connect to redis sentinel:", sentinel)
	}

	err = ValidateCurrentMaster()
	if err != nil {
		Fatal("Cannot switch to current master")
	}
	psc := redis.PubSubConn{c}
	Debug("Subscribing to sentinel (+switch-master).")
	psc.Subscribe("+switch-master")
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			Debug(fmt.Sprintf("%s: message: %s", v.Channel, v.Data))
			data := strings.Split(string(v.Data), string(' '))
			SwitchMaster(data[0], data[3], data[4])
		case redis.Subscription:
			Debug(fmt.Sprintf("%s: %s %d", v.Channel, v.Kind, v.Count))
		case error:
			Fatal("Error with redis connection:", psc)
		}
	}
}

func Run() {
	LoadTwemproxyConfig()
	SubscribeToSentinel()
}
