package main

func main() {
	app, err := BootstrapApp()
	if err != nil {
		panic(err)
	}
	app.WaitShutdown()
}
