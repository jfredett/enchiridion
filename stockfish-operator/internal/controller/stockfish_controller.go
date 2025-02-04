package controller

import (
	"context"
	"fmt"

	uuid "github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	sf "github.com/jfredett/enchridion/api/v1alpha1"
)

// StockfishReconciler reconciles a Stockfish object
type StockfishReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func newStockfishPod(cr *sf.Stockfish) *batchv1.Job {
	// This should spawn of my stockfish pods w/ the script in an environment var
	uuid := uuid.New().String()
	jobName := fmt.Sprintf("stockfish-analysis-%s", uuid)
	args := []string{
		"isready",
		fmt.Sprintf("position %s", cr.Spec.Position),
		"setoption name Hash value 256",
		fmt.Sprintf("setoption name MultiPV value %d", cr.Spec.Lines),
		fmt.Sprintf("go depth %d", cr.Spec.Depth),
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cr.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  jobName,
							Image: "docker-registry.emerald.city/stockfish-analyzer:latest",
							Args:  args,
						},
					},
				},
			},
		},
	}
}

// +kubebuilder:rbac:groups=job-runner.emerald.city,resources=stockfish,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=job-runner.emerald.city,resources=stockfish/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=job-runner.emerald.city,resources=stockfish/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *StockfishReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Stockfish resource
	var stockfish sf.Stockfish
	if err := r.Get(ctx, req.NamespacedName, &stockfish); err != nil {
		log.Error(err, "unable to fetch Stockfish")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info(fmt.Sprintf("Stockfish: %s", stockfish.Name))
	log.Info(fmt.Sprintf("Stockfish position command: %s", stockfish.Spec.Position))
	log.Info(fmt.Sprintf("Stockfish multipv option: setoption name MultiPV value %d", stockfish.Spec.Lines))
	log.Info(fmt.Sprintf("Stockfish go command: go depth %d", stockfish.Spec.Depth))

	switch stockfish.Spec.State {
	case "":
		pod := newStockfishPod(&stockfish)
		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "unable to create Stockfish pod")
			return ctrl.Result{}, err
		}

		stockfish.Spec.State = "Running"
		err := r.Status().Update(ctx, &stockfish)
		if err != nil {
			log.Error(err, "unable to update Stockfish status")
			return ctrl.Result{}, err
		}
	case "Running":
		// Monitor the status of the pod
		job := &batchv1.Job{}
		if err := r.Get(ctx, req.NamespacedName, job); err != nil {
			log.Error(err, "unable to fetch Job")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete {
				stockfish.Spec.State = "Completed"
				err := r.Status().Update(ctx, &stockfish)
				if err != nil {
					log.Error(err, "unable to update to completed status")
					return ctrl.Result{}, err
				}
			}
		}

	case "Completed":
		// Get the logs from the pod, parse them, and update the CRD, set status to CleanUp if this is successful,
		// failed otherwise
	case "Failed":
		// Copy the whole log to the CRD, set status to CleanUp
	case "CleanUp":
		// Delete the pod, set status to Done
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StockfishReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sf.Stockfish{}).
		Complete(r)
}
