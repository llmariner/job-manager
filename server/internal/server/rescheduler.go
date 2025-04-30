package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
)

const pullingImageReason = "Pulling"

// RunRescheduler requeues and reschedules the jobs.
func (s *S) RunRescheduler(ctx context.Context, interval, maxQueuedTime time.Duration) error {
	if err := s.rescheduleNotebooks(ctx, maxQueuedTime); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.rescheduleNotebooks(ctx, maxQueuedTime); err != nil {
				return ctx.Err()
			}
		}
	}
}

func (s *S) rescheduleNotebooks(ctx context.Context, maxQueuedTime time.Duration) error {
	s.logger.V(1).Info("Rescheduling notebooks ...")
	nbs, err := s.store.ListNotebooksByState(store.NotebookStateInitializing)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, nb := range nbs {
		// When a pod is in the stage of downloading image, it may take a long time before
		// it is ready to run. Avoid re-schedule the pod to another cluster.
		if strings.Contains(nb.Reason, pullingImageReason) {
			continue
		}
		if nb.UpdatedAt.Add(maxQueuedTime).Before(now) {
			nb.State = store.NotebookStateQueued
			nb.QueuedAction = store.NotebookQueuedActionRequeue
			if err := s.store.UpdateNotebookForRescheduling(nb); err != nil {
				return fmt.Errorf("requeue a notebook %s: %w", nb.NotebookID, err)
			}
			s.logger.V(1).Info("notebooked is to be requeued", "nb", nb)
		}
	}

	nbs, err = s.store.ListNotebooksByState(store.NotebookStateRequeued)
	if err != nil {
		return fmt.Errorf("reschedule notebooks: %w", err)
	}
	for _, nb := range nbs {
		nbProto, err := nb.V1Notebook()
		if err != nil {
			return fmt.Errorf("reschedule a notebook %s: %w", nb.NotebookID, err)
		}
		gpuCount := 0
		if nbProto.Resources != nil {
			gpuCount = int(nbProto.Resources.GpuCount)
		}
		sresult, err := s.scheduleNotebook(ctx, nb, gpuCount)
		if err != nil {
			// skip this notebook if it cannot be scheduled
			s.logger.Error(err, fmt.Sprintf("reschedule a notebook %s", nb.NotebookID))
			continue
		}
		nb.Reason = ""
		nb.State = store.NotebookStateQueued
		nb.QueuedAction = store.NotebookQueuedActionStart
		nb.ClusterID = sresult.ClusterID
		if err := nb.MutateMessage(func(p *v1.Notebook) {
			p.Status = string(store.NotebookStateQueued)
			p.ClusterId = sresult.ClusterID
			p.ClusterName = sresult.ClusterName
			p.KubernetesNamespace = sresult.Namespace
		}); err != nil {
			return err
		}
		if err := s.store.UpdateNotebookForRescheduling(nb); err != nil {
			return fmt.Errorf("reschedule a notebook %s: %w", nb.NotebookID, err)
		}
		s.logger.V(1).Info("notebook is rescheduled", "nb", nb)
	}
	return nil
}
