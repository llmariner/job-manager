package controller

import (
	"fmt"
	"os"
	"reflect"

	"github.com/awslabs/operatorpkg/context"
	v1 "github.com/llmariner/job-manager/api/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RemoteSyncerManager manages remote syncers.
type RemoteSyncerManager struct {
	sessionManagerEndpoint string

	ssClient       v1.SyncerServiceClient
	localK8sClient client.Client
}

// SetupWithManager sets up the controller with the Manager.
func (m *RemoteSyncerManager) SetupWithManager(mgr ctrl.Manager, ssClient v1.SyncerServiceClient, sessionManagerServerAddr string) error {
	m.ssClient = ssClient
	m.sessionManagerEndpoint = sessionManagerServerAddr
	m.localK8sClient = mgr.GetClient()
	return mgr.Add(m)
}

// Start starts the remote syncer manager.
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
		eg.Go(func() error {
			ctx := ctrl.LoggerInto(egCtx, log.WithName(c))
			rconf := getRestConfig(m.sessionManagerEndpoint, c, getAuthorizationToken())

			// TODO(aya): gracefully handle errors
			if err := newStatusSyncer(m.localK8sClient).start(ctx, rconf, i+1); err != nil {
				return fmt.Errorf("run status syncer %s: %s", c, err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Error(err, "Failed to run remote syncers")
		return fmt.Errorf("wait: %s", err)
	}
	return nil
}

func newStatusSyncer(localK8sClient client.Client) *statusSyncer {
	return &statusSyncer{
		localK8sClient: localK8sClient,
	}
}

// statusSyncer syncs the status of the remote job to the local job.
type statusSyncer struct {
	localK8sClient  client.Client
	remoteK8sClient client.Client
}

// start starts the status syncer.
func (s *statusSyncer) start(ctx context.Context, conf rest.Config, idx int) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting status syncer", "host", conf.Host)

	lbsl := labels.SelectorFromSet(labels.Set{jobLabelKey: controllerName})
	mgr, err := ctrl.NewManager(&conf, ctrl.Options{
		// TODO(aya): rethink the monitoring
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache:   cache.Options{DefaultLabelSelector: lbsl},
	})
	if err != nil {
		log.Error(err, "Failed to create manager")
		return fmt.Errorf("create manager: %s", err)
	}
	s.remoteK8sClient = mgr.GetClient()

	if err := ctrl.NewControllerManagedBy(mgr).
		Named(fmt.Sprintf("syncer%02d", idx)).
		For(&batchv1.Job{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return true },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		}).
		Complete(s); err != nil {
		log.Error(err, "Failed to setup syncer")
		return fmt.Errorf("setup syncer: %s", err)
	}

	log.Info("Starting manager...")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start manager: %s", err)
	}
	log.Info("Manager stopped")
	return nil
}

// Reconcile reconciles a local Job object and deploys it to the worker cluster.
func (s *statusSyncer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var remoteJob batchv1.Job
	if err := s.remoteK8sClient.Get(ctx, req.NamespacedName, &remoteJob); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get remote job")
		}
		return ctrl.Result{}, err
	}
	if !remoteJob.DeletionTimestamp.IsZero() {
		log.V(2).Info("Job is being deleted", "job", remoteJob.Name)
		return ctrl.Result{}, nil
	}

	var localJob batchv1.Job
	if err := s.localK8sClient.Get(ctx, req.NamespacedName, &localJob); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get local job")
		}
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(localJob.Status, remoteJob.Status) {
		log.V(4).Info("Status is up-to-date", "job", localJob.Name)
		return ctrl.Result{}, nil
	}
	patch := client.MergeFrom(&localJob)
	newJob := localJob.DeepCopy()
	newJob.Status = remoteJob.Status
	if err := s.localK8sClient.Status().Patch(ctx, newJob, patch); err != nil {
		log.Error(err, "Failed to update status", "job", localJob.Name)
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
