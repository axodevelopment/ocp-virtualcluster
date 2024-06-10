package controller

import (
	"context"
	organizationv1 "github.com/axodevelopment/ocp-virtualcluster/controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const (
	LabelKeyPart string = "organization/virtualcluster.name"
	//LabelKeyNSPart   string = "organization/virtualcluster.namespace"
	//DefaultNamespace string = "operator-virtualcluster"
	DefaultNamespace string = "virtualcluster-system"
	NodeLabelKeyPart string = "organization/virtualcluster."
	RetryInterval           = 100 * time.Second
)

func GetAppliedSelectorLabelKeyValue(ctx context.Context, vc *organizationv1.VirtualCluster) (string, string) {
	logger := log.FromContext(ctx)
	logger.Info("Generating Key-Value for labeling", "VirtualCluster", vc.Name)

	return NodeLabelKeyPart + vc.Name, vc.Name
}
