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

func create(x int32) *int32 {
	return &x
}

func newStockfishPod(cr *sf.Stockfish) *batchv1.Job {

	// TODO: Bundle the UUID into a jobname method and it should also set the UUID on the CRD? Or maybe just do everything by
	// name?
	jobName := fmt.Sprintf("stockfish-analysis-%s", uuid.New().String())

	// TODO: This array should be a metod on Stockfish
	args := []string{
		"isready",
		fmt.Sprintf("position %s", cr.Spec.Position),
		"setoption name Hash value 256",
		fmt.Sprintf("setoption name MultiPV value %d", cr.Spec.Lines),
		fmt.Sprintf("go depth %d", cr.Spec.Depth),
	}

	// TODO: I think this needs a finalizer to tell the controller when it's done so we can go grab it's output. Current
	// state has the operator never picking the thing back up

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:       jobName,
			Namespace:  cr.Namespace,
			Finalizers: []string{},
		},
		Spec: batchv1.JobSpec{
			// Ensures only one analysis at a time. Ideally this is externally configurable?
			Parallelism: create(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
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
	// Grab a handle to the logger
	log := logf.FromContext(ctx)

	log.Info(fmt.Sprintf("Stockfish ||| Reconciling Stockfish : Req: %s", req))

	// Fetch the Stockfish resource
	var stockfish sf.Stockfish
	if err := r.Get(ctx, req.NamespacedName, &stockfish); err != nil {
		log.Error(err, "unable to fetch Stockfish")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Based on the current state of the Stockfish resource, take the appropriate action
	switch stockfish.Status.State {

	case "": // The CRD is brand new and we don't have any pod running to analyze it, so start it up
		log.Info("Stockfish ||| Analysis is not running, starting")
		pod := newStockfishPod(&stockfish)

		//ctrl.SetControllerReference(&stockfish, pod, r.Scheme)

		// Update the state to 'Running'
		log.Info(fmt.Sprintf("Stockfish ||| Before: %+v", &stockfish))
		stockfish.Status.State = "Running"
		err := r.Status().Update(ctx, &stockfish)
		if err != nil {
			log.Error(err, "unable to update Stockfish status")
			return ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("Stockfish ||| After: %+v", &stockfish))
		log.Info("Stockfish ||| Updated Stockfish status to Running")

		// TODO: Configure finalizer
		pod.Finalizers = []string{"finalizers.emerald.city/stockfish-analysis"}

		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "unable to create Stockfish pod")
			return ctrl.Result{}, err
		}

	case "Running":
		log.Info("Stockfish ||| Analysis is running, checking for complete")
		// Monitor the status of the pod
		job := &batchv1.Job{}
		if err := r.Get(ctx, req.NamespacedName, job); err != nil {
			log.Error(err, "unable to fetch Job")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete {
				stockfish.Status.State = "Running"
				err := r.Status().Update(ctx, &stockfish)
				if err != nil {
					log.Error(err, "unable to update to completed status")
					return ctrl.Result{}, err
				}
			}
		}

	case "Completed":
		log.Info("Stockfish ||| Analysis is completed, parsing logs")
		// Get the logs from the pod, parse them, and update the CRD, set status to CleanUp if this is successful,
		// failed otherwise
	case "Failed":
		log.Info("Stockfish ||| Analysis failed, cleaning up")
		// Copy the whole log to the CRD, set status to CleanUp
	case "CleanUp":
		log.Info("Stockfish ||| Analysis is complete, cleaning up")
		// Delete the pod, set status to Done
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StockfishReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sf.Stockfish{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
