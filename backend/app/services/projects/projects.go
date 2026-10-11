package projects

import (
	"strings"

	"github.com/google/uuid"

	"ebox/app/facades"
	"ebox/app/models"
)

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List() ([]models.Project, error) {
	var list []models.Project
	if err := facades.Orm().Query().Order("name asc").Find(&list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Service) Find(uuidValue string) (*models.Project, error) {
	var project models.Project
	if err := facades.Orm().Query().Where("uuid", uuidValue).FirstOrFail(&project); err != nil {
		return nil, err
	}

	return &project, nil
}

func (s *Service) Create(name, description string) (*models.Project, error) {
	project := &models.Project{
		UUID:        uuid.NewString(),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	}

	if err := facades.Orm().Query().Create(project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Service) Update(project *models.Project, name, description string) error {
	project.Name = strings.TrimSpace(name)
	project.Description = strings.TrimSpace(description)

	return facades.Orm().Query().Save(project)
}

func (s *Service) Delete(project *models.Project) error {
	_, err := facades.Orm().Query().Delete(project)

	return err
}
