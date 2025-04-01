package controller

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	v1 "github.com/llmariner/job-manager/api/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"

	"github.com/llmariner/job-manager/syncer/internal/config"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
)

// Scheme defines methods for serializing and deserializing API objects.
var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(corev1.AddToScheme(Scheme))
	utilruntime.Must(jobsetv1alpha2.AddToScheme(Scheme))
	utilruntime.Must(batchv1.AddToScheme(Scheme))
}

// RemoteSyncerManager manages remote syncers.
type RemoteSyncerManager struct {
	sessionManagerEndpoint string

	ssClient       v1.SyncerServiceClient
	localK8sClient client.Client

	syncedKinds config.SyncedKindsConfig
}

// SetupWithManager sets up the controller with the Manager.
func (m *RemoteSyncerManager) SetupWithManager(
	mgr ctrl.Manager,
	ssClient v1.SyncerServiceClient,
	sessionManagerServerAddr string,
	syncedKinds config.SyncedKindsConfig,
) error {
	m.ssClient = ssClient
	m.sessionManagerEndpoint = sessionManagerServerAddr
	m.localK8sClient = mgr.GetClient()
	return mgr.Add(m)
}

// Start starts the remote syncer manager and blocks.
func (m *RemoteSyncerManager) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("syncer")
	log.Info("Starting remote syncer manager")

	// TODO(aya): dynamic cluster registration
	cls, err := m.ssClient.ListClusterIDs(
		appendAuthorization(ctx),
		&v1.ListClusterIDsRequest{})
	if err != nil {
		return fmt.Errorf("list clusters: %s", err)
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for i, c := range cls.Ids {
		log.Info("Starting remote syncer", "cluster", c)
		ctx := ctrl.LoggerInto(egCtx, log.WithName(c))
		rconf := getRestConfig(m.sessionManagerEndpoint, c, getAuthorizationToken())
		if m.syncedKinds.Jobs {
			eg.Go(func() error {
				return m.runStatusSyncer(ctx, rconf, c, i+1, jobControllerName, &batchv1.Job{}, syncJobsFn)
			})
		}

		if m.syncedKinds.JobSets {
			eg.Go(func() error {
				return m.runStatusSyncer(ctx, rconf, c, i+1, jobSetControllerName, &jobsetv1alpha2.JobSet{}, syncJobSetsFn)
			})
		}
	}

	if err := eg.Wait(); err != nil {
		log.Error(err, "Failed to run remote syncers")
		return fmt.Errorf("wait: %s", err)
	}
	return nil
}

// runStatusSyncer sets up a new remote job status syncer instance and starts reconciling. This method blocks until
// start errors or is aborted.
func (m *RemoteSyncerManager) runStatusSyncer(
	ctx context.Context,
	rconf rest.Config,
	clusterID string,
	idx int,
	controllerName string,
	object client.Object,
	reconcileFn clusterStatusObjectReconcileFn,
) error {
	typeName := reflect.TypeOf(object).Elem().Name()
	log := ctrl.LoggerFrom(ctx).
		WithValues("type", typeName, "idx", idx)
	log.Info("Starting status syncer", "host", rconf.Host)

	// TODO(aya): gracefully handle errors
	syncer := newStatusSyncer(m.localK8sClient, reconcileFn)
	instanceName := fmt.Sprintf("%s-syncer%02d", strings.ToLower(typeName), idx)
	mgr, err := initRemoteControllerManager(rconf, controllerName, instanceName, object, syncer)
	if err != nil {
		return fmt.Errorf("init %s remote controller %s: %w", typeName, clusterID, err)
	}
	syncer.remoteK8sClient = mgr.GetClient()

	log.Info("Starting manager...")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("run %s status syncer %s: %w", typeName, clusterID, err)
	}
	log.Info("Manager stopped")
	return nil
}

// abstract reconcile function as extension point
type clusterStatusObjectReconcileFn func(ctx context.Context, req ctrl.Request, remoteK8sClient, localK8sClient client.Client) (ctrl.Result, error)

// newStatusSyncer constructor
func newStatusSyncer(localK8sClient client.Client, fn clusterStatusObjectReconcileFn) *clusterStatusSyncer {
	return &clusterStatusSyncer{
		localK8sClient: localK8sClient,
		reconcileFn:    fn,
	}
}

// statusSyncer syncs the status of the remote src to the local src.
type clusterStatusSyncer struct {
	localK8sClient  client.Client
	remoteK8sClient client.Client
	reconcileFn     clusterStatusObjectReconcileFn
}

func (s *clusterStatusSyncer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return s.reconcileFn(ctx, req, s.remoteK8sClient, s.localK8sClient)
}

// initRemoteControllerManager setup the remote controller manager for this instance
func initRemoteControllerManager(
	conf rest.Config,
	controllerName string,
	instanceName string,
	managedObj client.Object,
	syncer *clusterStatusSyncer,
) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(&conf, ctrl.Options{
		Scheme: Scheme,
		// TODO(aya): rethink the monitoring
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache: cache.Options{
			DefaultLabelSelector: labels.SelectorFromSet(labels.Set{deployedByLabelKey: controllerName}),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create manager: %w", err)
	}
	if err := ctrl.NewControllerManagedBy(mgr).
		Named(instanceName).
		For(managedObj).
		WithEventFilter(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		}).
		Complete(syncer); err != nil {
		return nil, fmt.Errorf("setup controller: %w", err)
	}
	return mgr, nil
}

// syncJobsFn synchronizes the status of a remote Kubernetes src with its local counterpart.
func syncJobsFn(ctx context.Context, req ctrl.Request, remoteK8sClient, localK8sClient client.Client) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var remoteJob batchv1.Job
	if err := remoteK8sClient.Get(ctx, req.NamespacedName, &remoteJob); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get remote src")
		}
		return ctrl.Result{}, err
	}
	if !remoteJob.DeletionTimestamp.IsZero() {
		log.V(2).Info("Job is being deleted", "src", remoteJob.Name)
		return ctrl.Result{}, nil
	}

	var localJob batchv1.Job
	if err := localK8sClient.Get(ctx, req.NamespacedName, &localJob); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get local src")
		}
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(localJob.Status, remoteJob.Status) {
		log.V(4).Info("Status is up-to-date", "src", localJob.Name)
		return ctrl.Result{}, nil
	}
	patch := client.MergeFrom(&localJob)
	newJob := localJob.DeepCopy()
	newJob.Status = remoteJob.Status
	if err := localK8sClient.Status().Patch(ctx, newJob, patch); err != nil {
		log.Error(err, "Failed to update status", "src", localJob.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// syncJobSetsFn synchronizes the status of a remote Kubernetes jobSet with its local counterpart.
func syncJobSetsFn(ctx context.Context, req ctrl.Request, remoteK8sClient, localK8sClient client.Client) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var remoteJobSet jobsetv1alpha2.JobSet
	if err := remoteK8sClient.Get(ctx, req.NamespacedName, &remoteJobSet); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get remote jobSet")
		}
		return ctrl.Result{}, err
	}
	if !remoteJobSet.DeletionTimestamp.IsZero() {
		log.V(2).Info("JobSet is being deleted", "src", remoteJobSet.Name)
		return ctrl.Result{}, nil
	}

	var localJobSet jobsetv1alpha2.JobSet
	if err := localK8sClient.Get(ctx, req.NamespacedName, &localJobSet); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get local jobSet")
		}
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(localJobSet.Status, remoteJobSet.Status) {
		log.V(4).Info("Status is up-to-date", "jobSet", localJobSet.Name)
		return ctrl.Result{}, nil
	}
	patch := client.MergeFrom(&localJobSet)
	newJobSet := localJobSet.DeepCopy()
	newJobSet.Status = remoteJobSet.Status
	if err := localK8sClient.Status().Patch(ctx, newJobSet, patch); err != nil {
		log.Error(err, "Failed to update status", "jobSet", localJobSet.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func getRestConfig(endpoint, clusterID, token string) rest.Config {
	return rest.Config{
		Host:        fmt.Sprintf("%s/sessions/%s", endpoint, clusterID),
		BearerToken: token,
	}
}

func getAuthorizationToken() string {
	return os.Getenv("LLMARINER_SYNCER_API_KEY")
}

func appendAuthorization(ctx context.Context) context.Context {
	auth := fmt.Sprintf("Bearer %s", getAuthorizationToken())
	return metadata.AppendToOutgoingContext(ctx, "Authorization", auth)
}
