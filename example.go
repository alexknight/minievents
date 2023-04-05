package minevents

import "fmt"

const (
	DemoTest EventType = "DemoTest"
)


func DemoTestCallback(e Event) {
	fmt.Println(e)
}


func initEvents() {
	em := GetInstance()
	em.Start()
	em.AddListener(DemoTest, DemoTestCallback)
}

func closeEvents() {
	GetInstance().Stop()
}



func main() {
	defer func() {
		closeEvents()
	}()
	initEvents()
	GetInstance().SendEvent(Event{
		Type: DemoTest,
		Dict: map[string]interface{}{
			"key": "this is a demo",
		},
	})
}