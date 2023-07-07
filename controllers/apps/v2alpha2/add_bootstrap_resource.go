package v2alpha2

import (
	"context"

	emperror "emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv2alpha2 "github.com/emqx/emqx-operator/apis/apps/v2alpha2"
	innerReq "github.com/emqx/emqx-operator/internal/requester"
	"github.com/rory-z/go-hocon"
	"github.com/sethvargo/go-password/password"
)

type addBootstrap struct {
	*EMQXReconciler
}

func (a *addBootstrap) reconcile(ctx context.Context, instance *appsv2alpha2.EMQX, _ innerReq.RequesterInterface) subResult {
	for _, resource := range []client.Object{
		generateNodeCookieSecret(instance),
		generateBootstrapUserSecret(instance),
	} {
		if err := ctrl.SetControllerReference(instance, resource, a.Scheme); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to set controller reference")}
		}
		if err := a.Create(resource); err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				return subResult{err: emperror.Wrap(err, "failed to create bootstrap configMap")}
			}
		}
	}

	return subResult{}
}

func generateNodeCookieSecret(instance *appsv2alpha2.EMQX) *corev1.Secret {
	var cookie string

	config, _ := hocon.ParseString(instance.Spec.BootstrapConfig)
	cookie = config.GetString("node.cookie")
	if cookie == "" {
		cookie, _ = password.Generate(64, 10, 0, true, true)
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.NodeCookieNamespacedName().Name,
			Namespace: instance.Namespace,
			Labels:    instance.Labels,
		},
		StringData: map[string]string{
			"node_cookie": cookie,
		},
	}
}

func generateBootstrapUserSecret(instance *appsv2alpha2.EMQX) *corev1.Secret {
	bootstrapUsers := ""
	for _, apiKey := range instance.Spec.BootstrapAPIKeys {
		bootstrapUsers += apiKey.Key + ":" + apiKey.Secret + "\n"
	}

	defPassword, _ := password.Generate(64, 10, 0, true, true)
	bootstrapUsers += appsv2alpha2.DefaultBootstrapAPIKey + ":" + defPassword

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.BootstrapUserNamespacedName().Name,
			Namespace: instance.Namespace,
			Labels:    instance.Labels,
		},
		StringData: map[string]string{
			"bootstrap_user": bootstrapUsers,
		},
	}
}
