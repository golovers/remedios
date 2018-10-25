package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	pth := flag.String("f", "./", "config path")
	p := flag.String("p", "8080", "port")
	flag.Parse()

	ch := loadConf(*pth)
	conf := atomic.Value{}
	go func() {
		for {
			select {
			case c := <-ch:
				conf.Store(c)
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("%s - %s\n", r.Method, r.URL.Path)
		var v interface{}
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil && err != io.EOF {
			logrus.Errorf("parsing json the request failed: %v\n", err)
		}
		defer r.Body.Close()
		c := conf.Load().(*config)
		for _, e := range c.Endpoints {
			if e.Method == "" {
				e.Method = "get"
			}
			if e.Path == r.URL.Path && strings.ToLower(e.Method) == strings.ToLower(r.Method) {
				for _, c := range e.Cases {
					if reflect.DeepEqual(v, c.Request.Body) {
						if c.Response.Status == 0 {
							c.Response.Status = http.StatusOK
						}
						w.WriteHeader(c.Response.Status)
						v, err := json.Marshal(c.Response.Body)
						if err != nil {
							panic(err) // panic so that the owner update the config accordingly
						}
						w.Write(v)
						return
					}
				}

			}
		}

		// not found
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})
	if err := http.ListenAndServe(":"+*p, nil); err != nil {
		log.Fatalf("http: listen error: %v\n", err)
	}
}

type config struct {
	Endpoints []struct {
		Path   string
		Method string
		Cases  []struct {
			Request struct {
				Header map[string]string
				Body   interface{}
			}
			Response struct {
				Status int
				Body   interface{}
			}
		}
	}
}

func loadConf(path string) <-chan *config {
	ch := make(chan *config, 1)
	v := viper.New()
	name := "remedios"
	v.SetConfigType("json")
	v.AddConfigPath(path)
	v.SetConfigName(name)

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
	parse := func() error {
		var c config
		if err := v.Unmarshal(&c); err != nil {
			return err
		}
		ch <- &c
		return nil
	}
	if err := parse(); err != nil {
		panic(err)
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		logrus.Info("config changed...")
		if err := parse(); err != nil {
			logrus.Errorln("parsing error:", err)
		}
	})
	return ch
}
