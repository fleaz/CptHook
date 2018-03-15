package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1> Welcome on the status page</h1>")
}

func main() {

	// Load configuration frm file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var moduleList = viper.Sub("modules")

	// Status module
	if moduleList.GetBool("status.enabled") {
		log.Println("Status module is active")
		http.HandleFunc("/status", statusHandler)
	}

	// Start HTTP server
	log.Fatal(http.ListenAndServe(viper.GetString("http.listen"), nil))

}
