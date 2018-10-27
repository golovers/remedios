package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"crypto/md5"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	pth := flag.String("f", "./", "config path")
	p := flag.String("p", "8080", "port")
	flag.Parse()

	ch := loadConf(*pth)
	endponts := sync.Map{}
	go func() {
		for {
			select {
			case c := <-ch:
				endponts.Range(func(k, v interface{}) bool {
					endponts.Delete(k)
					return true
				})
				for _, e := range c.Endpoints {
					cases := make(map[string]kase)
					for _, c := range e.Cases {
						if c.Response.Status <= 0 {
							c.Response.Status = http.StatusOK
						}
						cases[hashit(c.Request.Body)] = c
					}
					endponts.Store(key(e.Method, e.Path), cases)
				}
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("%s - %s\n", r.Method, r.URL.Path)
		ev, ok := endponts.Load(key(r.Method, r.URL.Path))
		notfound := func() {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
			return
		}
		if !ok {
			notfound()
			return
		}
		c, ok := (ev.(map[string]kase))[hashit(r.Body)]
		if !ok {
			notfound()
			return
		}
		w.WriteHeader(c.Response.Status)
		v, err := json.Marshal(c.Response.Body)
		if err != nil {
			panic(err) // panic so that the owner update the config accordingly
		}
		w.Write(v)
	})
	if err := http.ListenAndServe(":"+*p, nil); err != nil {
		log.Fatalf("http: listen error: %v\n", err)
	}
}

type config struct {
	Endpoints []struct {
		Path   string
		Method string
		Cases  []kase
	}
}

type kase struct {
	Request struct {
		Header map[string]string
		Body   interface{}
	}
	Response struct {
		Status int
		Body   interface{}
	}
}

func loadConf(path string) <-chan *config {
	ch := make(chan *config, 1)
	v := viper.New()
	v.SetConfigType("json")
	v.AddConfigPath(path)
	v.SetConfigName("remedios")

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

func key(method, path string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(method), strings.ToLower(path))
}

func hashit(v interface{}) string {
	if v == nil {
		return ""
	}
	h := func(v interface{}) string {
		h := md5.New()
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		h.Write(b)
		return base64.StdEncoding.EncodeToString(h.Sum(nil))
	}
	if rc, ok := v.(io.ReadCloser); ok {
		var val interface{}
		// need to do this to make sure both keys use the same encoding & decoding
		if err := json.NewDecoder(rc).Decode(&val); err != nil {
			return ""
		}
		return h(val)
	}
	return h(v)
}
