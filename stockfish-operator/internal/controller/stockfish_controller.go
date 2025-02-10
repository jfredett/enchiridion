package controller

import (
	"context"
	"fmt"

	uuid "github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	// TODO: This array should be a method on Stockfish
	args := []string{
		"isready",
		"setoption name Hash value 256",
		fmt.Sprintf("setoption name MultiPV value %d", cr.Spec.Lines),
		"uci",
		fmt.Sprintf("position %s", cr.Spec.Position),
		fmt.Sprintf("go depth %d", cr.Spec.Depth),
	}

	// TODO: Method on stockfish

	// TODO: I think this needs a finalizer to tell the controller when it's done so we can go grab it's output. Current
	// state has the operator never picking the thing back up

	// TODO: Name should be something like `analysis-<name of CRD>"? I thought UID but that seems less convenient.

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:       cr.Status.JobName,
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
							Name:            cr.Status.JobName,
							Image:           "docker-registry.emerald.city/stockfish-analyzer:latest",
							ImagePullPolicy: corev1.PullAlways,
							Args:            args,
							Env: []corev1.EnvVar{
								{
									// TODO: would be ideal to pull this from the Endpoint API dynamically
									Name:  "REDIS_URL",
									Value: "redis-master",
								},
								{
									Name:  "UUID",
									Value: cr.Status.Analysis,
								},
							},
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
	// TODO: Handle Delete events better?

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

	// TODO: break out into methods
	case "": // The CRD is brand new and we don't have any pod running to analyze it, so start it up
		log.Info("Stockfish ||| Analysis is not running, starting")

		// Update the status, and note the job name (this may not be necessary? FIXME)
		stockfish.Status.State = "Running"
		uuid := fmt.Sprintf("%s", uuid.New().String())
		stockfish.Status.Analysis = uuid
		stockfish.Status.JobName = fmt.Sprintf("stockfish-analysis-%s", uuid)

		pod := newStockfishPod(&stockfish)

		err := r.Status().Update(ctx, &stockfish)
		if err != nil {
			log.Error(err, "unable to update Stockfish status")
			return ctrl.Result{}, err
		}

		// TODO: Configure finalizer? I don't want the job to be cleaned until I get to it
		// pod.Finalizers = []string{"finalizers.emerald.city/stockfish-analysis"}

		log.Info("Stockfish ||| Starting Analysis Job ...")
		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "unable to create Stockfish pod")
			return ctrl.Result{}, err
		}
		log.Info("Stockfish ||| ... Started")

	case "Running":
		log.Info("Stockfish ||| Analysis is running, checking for complete")

		// TODO: Maybe move this to above the switch, and set a flag if it's not there?
		job := &batchv1.Job{}
		jobName := types.NamespacedName{
			Namespace: stockfish.Namespace,
			Name:      stockfish.Status.JobName,
		}
		if err := r.Get(ctx, jobName, job); err != nil {
			log.Error(err, "unable to fetch Job")
			return ctrl.Result{Requeue: true}, client.IgnoreNotFound(err)
		}
		log.Info(fmt.Sprintf("Stockfish ||| Retrieved Job: %+v", job))

		log.Info("Stockfish ||| Checking Job Conditions")
		// if we have conditions, check them
		if job.Status.Conditions != nil {
			for _, condition := range job.Status.Conditions {
				log.Info(fmt.Sprintf("Stockfish |||||| Condition: %+v", condition))
				log.Info(fmt.Sprintf("Stockfish |||||| Condition.Type: %+v", condition.Type))
				if condition.Type == batchv1.JobComplete {
					log.Info("Stockfish ||| Analysis is complete, marking")
					stockfish.Status.State = "Completed"

					err := r.Status().Update(ctx, &stockfish)
					if err != nil {
						log.Error(err, "unable to update to completed status")
						return ctrl.Result{}, err
					}
				}
			}
		} else { // requeue
			return ctrl.Result{Requeue: true}, nil
		}

	case "Completed":
		log.Info("Stockfish ||| Analysis is completed, parsing logs")

		// Update the CRD w/ the results of the analysis from the pod.
		//// grab the pod
		job := &batchv1.Job{}
		jobName := types.NamespacedName{
			Namespace: stockfish.Namespace,
			Name:      stockfish.Status.JobName,
		}
		if err := r.Get(ctx, jobName, job); err != nil {
			log.Error(err, "unable to fetch Pod")
			return ctrl.Result{Requeue: true}, client.IgnoreNotFound(err)
		}
		//// Grab the Logs from whereever I stashed them, filed by some UID

		//// attach the output to the CRD
		stockfish.Status.State = "CleanUp"
		stockfish.Status.Analysis = "TODO!"
		if err := r.Status().Update(ctx, &stockfish); err != nil {
			log.Error(err, "unable to update Stockfish")
			return ctrl.Result{}, err
		}

	case "CleanUp":
		log.Info("Stockfish ||| Cleaning up")

		log.Info("Stockfish ||| Getting Job for Cleanup")
		job := &batchv1.Job{}
		jobName := types.NamespacedName{
			Namespace: stockfish.Namespace,
			Name:      stockfish.Status.JobName,
		}
		if err := r.Get(ctx, jobName, job); err != nil {
			log.Error(err, "unable to fetch Pod")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Info("Stockfish ||| Deleting Job")
		if err := r.Delete(ctx, job); err != nil {
			log.Error(err, "unable to delete Job")
			return ctrl.Result{Requeue: true}, err
		}
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
