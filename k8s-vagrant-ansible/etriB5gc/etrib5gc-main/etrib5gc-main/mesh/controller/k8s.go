package controller

import (
	"context"
	"etrib5gc/mesh/models"
	"time"

	"github.com/reogac/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func (c *Controller) loadK8sServices() (err error) {
	// Step  1: List current services
	svcList, err := c.k8sCli.CoreV1().Services(c.k8sNameSpace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		err = utils.WrapError("List services", err)
		return
	}

	log.Infof("Current Services:")
	for _, svc := range svcList.Items {
		log.Infof("  - %s\n", svc.Name)
	}

	// Step 2: Watch for changes
	log.Infof("Watching for Service changes...")

	if c.svcWatcher, err = c.k8sCli.CoreV1().Services(c.k8sNameSpace).Watch(context.TODO(), metav1.ListOptions{}); err != nil {
		err = utils.WrapError("Start watch", err)
		return
	}
	go c.watchK8sServices()
	return
}

func (c *Controller) watchK8sServices() {
	for event := range c.svcWatcher.ResultChan() {
		service, ok := event.Object.(*corev1.Service)
		if !ok {
			log.Infof("receive unexpected type")
			continue
		}

		switch event.Type {
		case watch.Added:
			log.Infof("[ADDED]    %s at %s\n", service.Name, service.CreationTimestamp.Time.Format(time.RFC3339))
			c.sman.handleUpdateService([]models.ServiceDefinition{
				{
					Id:        service.Name,
					Selectors: service.Spec.Selector,
					Stateful:  true,
				},
			})
		case watch.Modified:
			log.Infof("[MODIFIED] %s\n", service.Name)
			c.sman.handleUpdateService([]models.ServiceDefinition{
				{
					Id:        service.Name,
					Selectors: service.Spec.Selector,
					Stateful:  true,
				},
			})

		case watch.Deleted:
			log.Infof("[DELETED]  %s\n", service.Name)
		default:
			log.Infof("[UNKNOWN]  %s\n", service.Name)
		}
	}

	log.Infof("Watcher closed.")

}
