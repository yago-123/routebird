package bgpalloc

import (
	"context"
	"fmt"
	bgpv1alphav1 "github.com/yago-123/routebird/api/v1alphav1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type BGPAllocReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *BGPAllocReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var routeCR bgpv1alphav1.BGPRoute
	if err := r.Get(ctx, req.NamespacedName, &routeCR); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// todo(): reconstruct the state all the time is alloc and space expensive
	allocatedIPs := make(map[string]bool)

	// todo: This piece of code is testing only. IP allocation should be done in a way that accepts large sets
	// todo: of IPs. Right now will crash with a simple /124 range in IPv6
	// Expand the IP ranges to individual IPs
	allocatedIPs, err := expandAllocatableIPRanges(routeCR.Spec.AllocatableIPRanges)
	if err != nil {
		logger.Error(err, "Failed to expand allocatable IP ranges")
		return ctrl.Result{}, err
	}

	var services corev1.ServiceList
	if errListing := r.List(ctx, &services); errListing != nil {
		logger.Error(errListing, "Failed to list Services")
		return ctrl.Result{}, errListing
	}

	// Mark IPs that are already in use by services
	// todo: ideally, this should be tracked and stored instead of recomputing all the time
	markUsedIPsFromServices(&services, allocatedIPs)

	// Assign IPs to services that do not have an IP yet
	for i := range services.Items {
		svc := &services.Items[i]

		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			continue
		}

		// Skip if already has an IP
		hasIP := false
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				hasIP = true
				break
			}
		}
		if hasIP {
			continue
		}

		// todo(): inefficient aswell
		// Find a free IP
		freeIP := retrieveFreeIP(allocatedIPs)
		if freeIP == "" {
			logger.Info("No free IPs available for service", "service", svc.Name)
			continue
		}

		// Patch the service status with the new IP
		svcPatch := svc.DeepCopy()
		svcPatch.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
			{IP: freeIP},
		}

		if errPatch := r.Status().Patch(ctx, svcPatch, client.MergeFrom(svc)); errPatch != nil {
			logger.Error(errPatch, "Failed to patch service with allocated IP", "ip", freeIP)
			return ctrl.Result{}, errPatch
		}

		logger.Info("Assigned IP to service", "service", svc.Name, "ip", freeIP)

		// Mark IP as used
		allocatedIPs[freeIP] = true
	}

	return ctrl.Result{}, nil
}

// expandAllocatedIPRanges expands a list of IP ranges into a map of IPs
func expandAllocatableIPRanges(ranges []string) (map[string]bool, error) {
	allocated := make(map[string]bool)
	for _, ipRange := range ranges {
		ips, err := parseIPRange(ipRange)
		if err != nil {
			return nil, fmt.Errorf("parsing range %s: %w", ipRange, err)
		}
		for _, ip := range ips {
			allocated[ip.String()] = false
		}
	}
	return allocated, nil
}

// parseIPRange marks as allocated the IPs already in use by services
func markUsedIPsFromServices(services *corev1.ServiceList, allocated map[string]bool) {
	for _, svc := range services.Items {
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			continue
		}
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP == "" {
				continue
			}

			allocated[ingress.IP] = true
		}
	}
}

// retrieveFreeIP retrieves a free IP from the allocated map
func retrieveFreeIP(allocated map[string]bool) string {
	for ip, used := range allocated {
		if !used {
			return ip
		}
	}

	return ""
}

func (r *BGPAllocReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				svc, ok := e.Object.(*corev1.Service)
				return ok && svc.Spec.Type == corev1.ServiceTypeLoadBalancer
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				newSvc, ok := e.ObjectNew.(*corev1.Service)
				return ok && newSvc.Spec.Type == corev1.ServiceTypeLoadBalancer
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				svc, ok := e.Object.(*corev1.Service)
				return ok && svc.Spec.Type == corev1.ServiceTypeLoadBalancer
			},
		}).
		Complete(r)
}
