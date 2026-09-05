// Package repositories provides thin data-access wrappers around GORM for
// each aggregate. Handlers and services depend on these instead of talking
// to *gorm.DB directly, keeping query logic in one place per entity.
package repositories

import "gorm.io/gorm"

type Repositories struct {
	Admins      *AdminRepository
	Users       *VPNUserRepository
	Devices     *DeviceRepository
	Servers     *ServerRepository
	AccessKeys  *AccessKeyRepository
	AuditLogs   *AuditLogRepository
	Metrics     *ServerMetricRepository
}

func New(db *gorm.DB) *Repositories {
	return &Repositories{
		Admins:     &AdminRepository{db: db},
		Users:      &VPNUserRepository{db: db},
		Devices:    &DeviceRepository{db: db},
		Servers:    &ServerRepository{db: db},
		AccessKeys: &AccessKeyRepository{db: db},
		AuditLogs:  &AuditLogRepository{db: db},
		Metrics:    &ServerMetricRepository{db: db},
	}
}

type Pagination struct {
	Page     int
	PageSize int
}

func (p Pagination) normalize() (page, pageSize, offset int) {
	page = p.Page
	if page < 1 {
		page = 1
	}
	pageSize = p.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset = (page - 1) * pageSize
	return
}
