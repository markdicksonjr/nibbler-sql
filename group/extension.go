package nibbler_user_group

import (
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/markdicksonjr/nibbler"
	nibbler_sql "github.com/markdicksonjr/nibbler-sql"
	"github.com/markdicksonjr/nibbler/user/group"
	"github.com/thoas/go-funk"
)

type SqlPersistenceExtension struct {
	nibbler_user_group.PersistenceExtension
	SqlExtension *nibbler_sql.Extension
	tx *gorm.DB
}

func (s *SqlPersistenceExtension) StartTransaction() (nibbler_user_group.PersistenceExtension, error) {
	tx := s.SqlExtension.Db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &SqlPersistenceExtension{
		SqlExtension: s.SqlExtension,
		tx:           tx,
	}, nil
}

func (s *SqlPersistenceExtension) RollbackTransaction() error {
	s.tx.Rollback()
	s.tx = nil
	return nil
}

func (s *SqlPersistenceExtension) CommitTransaction() error {
	err := s.tx.Commit().Error
	s.tx = nil
	return err
}

func (s *SqlPersistenceExtension) GetPrivilegesForAction(
	groupId string,
	resourceId *string,
	action string,
) ([]nibbler.GroupPrivilege, error) {
	var db = s.SqlExtension.Db
	if s.tx != nil {
		db = s.tx
	}

	var privs []nibbler.GroupPrivilege
	if resourceId == nil {
		err := db.Model(&nibbler.Group{ID: groupId}).
			Where(&nibbler.GroupPrivilege{Action: action}).
			Related(&privs).
			Error
		return privs, err
	}
	err := db.Model(&nibbler.Group{ID: groupId}).
		Where(&nibbler.GroupPrivilege{Action: action, ResourceID: *resourceId}).
		Related(&privs).
		Error
	return privs, err
}

func (s *SqlPersistenceExtension) GetGroupMembershipsForUser(userId string) ([]nibbler.GroupMembership, error) {
	var db = s.SqlExtension.Db
	if s.tx != nil {
		db = s.tx
	}

	var memberships []nibbler.GroupMembership
	if err := db.Where(&nibbler.GroupMembership{MemberID: userId}).Find(&memberships).Error; err != nil {
		return nil, err
	}
	return memberships, nil
}

func (s *SqlPersistenceExtension) SetGroupMembership(groupId string, userId string, role string) (nibbler.GroupMembership, error) {
	var db = s.SqlExtension.Db
	if s.tx != nil {
		db = s.tx
	}

	groupMembership := nibbler.GroupMembership{
		ID:       uuid.New().String(),
		GroupID:  groupId,
		MemberID: userId,
		Role:     role,
	}
	err := db.Where(&nibbler.GroupMembership{GroupID: groupId, MemberID: userId}).Save(&groupMembership).Error
	return groupMembership, err
}

func (s *SqlPersistenceExtension) CreateGroup(group nibbler.Group) error {
	return s.SqlExtension.Db.Create(group).Error
}

func (s *SqlPersistenceExtension) GetGroupsById(groupIds []string) ([]nibbler.Group, error) {
	var groups []nibbler.Group
	err := s.SqlExtension.Db.Where("id IN (?)", funk.UniqString(groupIds)).Find(&groups).Error
	return groups, err
}
func (s *SqlPersistenceExtension) AddPrivilegeToGroups(groupIdList []string, targetGroupId string, action string) error {
	groups, err := s.GetGroupsById(groupIdList)
	if err != nil {
		return err
	}

	// TODO: add only if undefined?
	for _, g := range groups {
		newPriv := nibbler.GroupPrivilege{
			ID:         uuid.New().String(),
			GroupID:    g.ID,
			ResourceID: targetGroupId,
			Action:     action,
		}

		if s.tx != nil {
			if err := s.tx.Save(&newPriv).Error; err != nil {
				return err
			}
		} else {
			if err := s.SqlExtension.Db.Save(&newPriv).Error; err != nil {
				return err
			}
		}

	}
	return nil
}