package gormdb

////for test
//type BaseRepository struct {
//	db *gorm.DB
//}
//
//func NewBaseRepository(db *gorm.DB) *BaseRepository {
//	return &BaseRepository{db: db}
//}
//
//func (r *BaseRepository) DB() *gorm.DB {
//	//todo from thread local
//	r.db.Transaction(func(tx *gorm.DB) error {
//
//		return nil
//	})
//	return r.db
//}
//
//func (r *BaseRepository) Transaction(fc func(tx *gorm.DB) error) error {
//
//	return r.DB().Transaction(fc)
//}

