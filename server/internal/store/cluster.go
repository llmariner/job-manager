package store

import (
	"errors"

	"gorm.io/gorm"
)

// Cluster represents the cluster.
type Cluster struct {
	gorm.Model

	ClusterID string `gorm:"uniqueIndex"`

	TenantID string `gorm:"index"`

	// Status is a marshalled proto message ClusterStatus.
	Status []byte
}

// CreateOrUpdateCluster creates or updates a cluster.
func (s *S) CreateOrUpdateCluster(c *Cluster) error {
	var existing Cluster
	if err := s.db.Where("cluster_id = ?", c.ClusterID).Take(&existing).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// No existing record. Create a new one.
		if err := s.db.Create(c).Error; err != nil {
			return err
		}
		return nil
	}

	// Found an existing record. Update it.
	existing.Status = c.Status
	if err := s.db.Save(&existing).Error; err != nil {
		return err
	}

	return nil
}

// GetClusterByID gets a cluster by its ID.
func (s *S) GetClusterByID(clusterID string) (*Cluster, error) {
	var c Cluster
	if err := s.db.Where("cluster_id = ?", clusterID).Take(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ListClustersByTenantID lists clusters by tenant ID.
func (s *S) ListClustersByTenantID(tenantID string) ([]*Cluster, error) {
	var clusters []*Cluster
	if err := s.db.Where("tenant_id = ?", tenantID).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}
