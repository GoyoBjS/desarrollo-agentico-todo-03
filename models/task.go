package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// Valores admitidos para la prioridad de una tarea.
const (
	PriorityHigh   = "alta"
	PriorityMedium = "media"
	PriorityLow    = "baja"
)

var priorityLabels = map[string]string{
	PriorityHigh:   "Alta",
	PriorityMedium: "Media",
	PriorityLow:    "Baja",
}

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Priority  string    `json:"priority" db:"priority" form:"priority"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// BeforeValidate asigna la prioridad media cuando la tarea no trae ninguna.
func (t *Task) BeforeValidate(tx *pop.Connection) error {
	t.Priority = strings.TrimSpace(t.Priority)
	if t.Priority == "" {
		t.Priority = PriorityMedium
	}
	return nil
}

// PendingCount devuelve cuántas tareas de la colección no están completadas.
func (t Tasks) PendingCount() int {
	count := 0
	for _, task := range t {
		if !task.Completed {
			count++
		}
	}
	return count
}

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	if _, ok := priorityLabels[t.Priority]; !ok {
		verrs.Add("priority", "La prioridad no es válida.")
	}

	return verrs, nil
}

// PriorityLabel devuelve el nombre visible de la prioridad de la tarea.
func (t Task) PriorityLabel() string {
	return priorityLabels[t.Priority]
}
