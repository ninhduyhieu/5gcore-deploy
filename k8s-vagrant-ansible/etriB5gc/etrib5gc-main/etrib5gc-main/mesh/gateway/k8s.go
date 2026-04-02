package gateway

import (
	"context"
	"fmt"
	"github.com/reogac/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// update endpoint IP and labels from K8s control plane
func (gw *Gateway) updateEpInfo(podName string, nameSpace string, ep *Endpoint) error {
	if len(nameSpace) > 0 && nameSpace != gw.k8sNameSpace {
		return fmt.Errorf("Mismatch namespace")
	}

	pod, err := gw.k8sCli.CoreV1().Pods(gw.k8sNameSpace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return utils.WrapError("Get Pod's metadata", err)
	}
	for k, v := range pod.Labels {
		ep.labels[k] = v
	}
	/*
		if ep.ip = net.ParseIP(pod.Status.PodIP); len(ep.ip) == 0 {
			return fmt.Errorf("Fail to get Pod IP from K8s control plane")
		} else {
			gw.Infof("Endpoint send IP:%s", pod.Status.PodIP)
		}
	*/
	return nil
}
