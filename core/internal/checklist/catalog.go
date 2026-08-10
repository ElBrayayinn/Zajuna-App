package checklist

import "fmt"

// Status is the user-facing state of a checklist item.
type Status string

const (
	StatusYes     Status = "SI"
	StatusNo      Status = "NO"
	StatusPending Status = "PENDIENTE"
)

type Category struct {
	ID        string
	Code      string
	Label     string
	SortOrder int
}

type Item struct {
	ID           string
	CategoryCode string
	ItemCode     string
	Description  string
	MaxEvidences int
	SortOrder    int
	GroupName    string
}

func ValidStatus(value string) bool {
	return Status(value) == StatusYes || Status(value) == StatusNo || Status(value) == StatusPending
}

func NormalizeStatus(value string) Status {
	switch value {
	case string(StatusYes):
		return StatusYes
	case string(StatusNo):
		return StatusNo
	default:
		return StatusPending
	}
}

func Categories() []Category {
	return []Category{
		{ID: "cat-1", Code: "1", Label: "1. Cronograma", SortOrder: 1},
		{ID: "cat-2", Code: "2", Label: "2. Perfil", SortOrder: 2},
		{ID: "cat-3", Code: "3", Label: "3. Disponibilidad", SortOrder: 3},
		{ID: "cat-4", Code: "4", Label: "4. Menú del Curso", SortOrder: 4},
		{ID: "cat-5", Code: "5", Label: "5. Calificaciones", SortOrder: 5},
		{ID: "cat-6", Code: "6", Label: "6. Configuración", SortOrder: 6},
		{ID: "cat-7", Code: "7", Label: "7. Seguimiento", SortOrder: 7},
		{ID: "cat-8", Code: "8", Label: "8. Sesiones en Línea", SortOrder: 8},
		{ID: "cat-9", Code: "9", Label: "9. Foros", SortOrder: 9},
		{ID: "cat-10", Code: "10", Label: "10. Evidencias", SortOrder: 10},
		{ID: "cat-11", Code: "11", Label: "11. Anuncios", SortOrder: 11},
		{ID: "cat-12", Code: "12", Label: "12. Grabaciones", SortOrder: 12},
		{ID: "cat-13", Code: "13", Label: "13. Documentos", SortOrder: 13},
		{ID: "cat-14", Code: "14", Label: "14. Conclusión Foros", SortOrder: 14},
		{ID: "cat-15", Code: "15", Label: "15. Netiqueta", SortOrder: 15},
	}
}

func Items() []Item {
	return []Item{
		{ID: "item-001", CategoryCode: "1", ItemCode: "1.1.1", Description: "Cronograma General - Nombre de las Fases", MaxEvidences: 1, SortOrder: 1, GroupName: "cronograma_general"},
		{ID: "item-002", CategoryCode: "1", ItemCode: "1.1.2", Description: "Cronograma General - Actividades de proyecto", MaxEvidences: 1, SortOrder: 2, GroupName: "cronograma_general"},
		{ID: "item-003", CategoryCode: "1", ItemCode: "1.1.3", Description: "Cronograma General - Actividades de aprendizaje", MaxEvidences: 1, SortOrder: 3, GroupName: "cronograma_general"},
		{ID: "item-004", CategoryCode: "1", ItemCode: "1.1.4", Description: "Cronograma General - Tiempo de duración estimado", MaxEvidences: 1, SortOrder: 4, GroupName: "cronograma_general"},
		{ID: "item-005", CategoryCode: "1", ItemCode: "1.1.5", Description: "Cronograma General - Fecha de inicio y fin de cada fase", MaxEvidences: 1, SortOrder: 5, GroupName: "cronograma_general"},
		{ID: "item-006", CategoryCode: "1", ItemCode: "1.2.1", Description: "Cronograma de Fase Vigente - Nombre de la Fase", MaxEvidences: 13, SortOrder: 6, GroupName: "cronograma_vigente"},
		{ID: "item-007", CategoryCode: "1", ItemCode: "1.2.2", Description: "Cronograma de Fase Vigente - Nombre de las actividades de proyecto", MaxEvidences: 5, SortOrder: 7, GroupName: "cronograma_vigente"},
		{ID: "item-008", CategoryCode: "1", ItemCode: "1.2.3", Description: "Cronograma de Fase Vigente - Actividades de aprendizaje a realizar", MaxEvidences: 5, SortOrder: 8, GroupName: "cronograma_vigente"},
		{ID: "item-009", CategoryCode: "1", ItemCode: "1.2.4", Description: "Cronograma de Fase Vigente - Resultados de Aprendizaje", MaxEvidences: 5, SortOrder: 9, GroupName: "cronograma_vigente"},
		{ID: "item-010", CategoryCode: "1", ItemCode: "1.2.5", Description: "Cronograma de Fase Vigente - Fecha de inicio y finalización de actividades", MaxEvidences: 5, SortOrder: 10, GroupName: "cronograma_vigente"},
		{ID: "item-011", CategoryCode: "1", ItemCode: "1.2.6", Description: "Cronograma de Fase Vigente - Evidencias a presentar", MaxEvidences: 5, SortOrder: 11, GroupName: "cronograma_vigente"},
		{ID: "item-012", CategoryCode: "1", ItemCode: "1.2.7", Description: "Cronograma de Fase Vigente - Instructor/Área responsable", MaxEvidences: 5, SortOrder: 12, GroupName: "cronograma_vigente"},
		{ID: "item-013", CategoryCode: "2", ItemCode: "2.1.1", Description: "Perfil del Instructor - Información académica y experiencia laboral", MaxEvidences: 1, SortOrder: 13, GroupName: "perfil_instructor"},
		{ID: "item-014", CategoryCode: "2", ItemCode: "2.1.2", Description: "Perfil del Instructor - Correo electrónico institucional", MaxEvidences: 1, SortOrder: 14, GroupName: "perfil_instructor"},
		{ID: "item-015", CategoryCode: "2", ItemCode: "2.1.3", Description: "Perfil del Instructor - Regional y Centro de Formación", MaxEvidences: 1, SortOrder: 15, GroupName: "perfil_instructor"},
		{ID: "item-016", CategoryCode: "2", ItemCode: "2.1.4", Description: "Perfil del Instructor - Horario de atención sincrónica y ruta/enlace", MaxEvidences: 1, SortOrder: 16, GroupName: "perfil_instructor"},
		{ID: "item-017", CategoryCode: "2", ItemCode: "2.1.5", Description: "Perfil del Instructor - Área a orientar dentro del programa", MaxEvidences: 1, SortOrder: 17, GroupName: "perfil_instructor"},
		{ID: "item-018", CategoryCode: "3", ItemCode: "3.1", Description: "Disponibilidad del material de trabajo y enlaces de envío de evidencias", MaxEvidences: 5, SortOrder: 18, GroupName: "disponibilidad"},
		{ID: "item-019", CategoryCode: "4", ItemCode: "4.1", Description: "Menú del Curso organizado con las secciones estipuladas", MaxEvidences: 1, SortOrder: 19, GroupName: "menu_curso"},
		{ID: "item-020", CategoryCode: "5", ItemCode: "5.1", Description: "Asociación de actividades y evidencias en el espacio Calificaciones", MaxEvidences: 5, SortOrder: 20, GroupName: "calificaciones"},
		{ID: "item-021", CategoryCode: "6", ItemCode: "6.1", Description: "Configuración de fecha límite de entrega en cada evidencia", MaxEvidences: 5, SortOrder: 21, GroupName: "configuracion"},
		{ID: "item-022", CategoryCode: "7", ItemCode: "7.1.1", Description: "Sección Seguimiento y Evaluación oculta: Subsección Reporte de Curso", MaxEvidences: 1, SortOrder: 22, GroupName: "seguimiento_evaluacion"},
		{ID: "item-023", CategoryCode: "7", ItemCode: "7.1.2", Description: "Sección Seguimiento y Evaluación oculta: Subsección Seguimiento a la Formación", MaxEvidences: 1, SortOrder: 23, GroupName: "seguimiento_evaluacion"},
		{ID: "item-024", CategoryCode: "7", ItemCode: "7.2", Description: "Subsección Reporte de Curso organizada por cada fase del programa", MaxEvidences: 1, SortOrder: 24, GroupName: "seguimiento_evaluacion"},
		{ID: "item-025", CategoryCode: "7", ItemCode: "7.3.1", Description: "Subsección Seguimiento a la Formación - Comités evaluativos – Actas", MaxEvidences: 1, SortOrder: 25, GroupName: "seguimiento_documentos"},
		{ID: "item-026", CategoryCode: "7", ItemCode: "7.3.2", Description: "Subsección Seguimiento a la Formación - Documentos de retención", MaxEvidences: 1, SortOrder: 26, GroupName: "seguimiento_documentos"},
		{ID: "item-027", CategoryCode: "7", ItemCode: "7.3.3", Description: "Subsección Seguimiento a la Formación - Reuniones EEF – Actas", MaxEvidences: 1, SortOrder: 27, GroupName: "seguimiento_documentos"},
		{ID: "item-028", CategoryCode: "7", ItemCode: "7.4.1", Description: "Subsección Comités evaluativos-Actas - Actas de Comité", MaxEvidences: 1, SortOrder: 28, GroupName: "seguimiento_documentos"},
		{ID: "item-029", CategoryCode: "7", ItemCode: "7.4.2", Description: "Subsección Comités evaluativos-Actas - Planes de Mejoramiento", MaxEvidences: 1, SortOrder: 29, GroupName: "seguimiento_documentos"},
		{ID: "item-030", CategoryCode: "7", ItemCode: "7.4.3", Description: "Subsección Comités evaluativos-Actas - Registro de Novedades", MaxEvidences: 1, SortOrder: 30, GroupName: "seguimiento_documentos"},
		{ID: "item-031", CategoryCode: "7", ItemCode: "7.4.4", Description: "Subsección Comités evaluativos-Actas - Llamados de Atención", MaxEvidences: 1, SortOrder: 31, GroupName: "seguimiento_documentos"},
		{ID: "item-032", CategoryCode: "8", ItemCode: "8.1", Description: "Organización de la sección Sesiones en Línea en el menú", MaxEvidences: 1, SortOrder: 32, GroupName: "sesiones_linea"},
		{ID: "item-033", CategoryCode: "8", ItemCode: "8.2", Description: "Organización de Sesiones en Línea / Subsecciones por Fase", MaxEvidences: 1, SortOrder: 33, GroupName: "sesiones_linea"},
		{ID: "item-034", CategoryCode: "8", ItemCode: "8.3", Description: "Organización de Sesiones en Línea / Subsecciones por Fase y Mes", MaxEvidences: 1, SortOrder: 34, GroupName: "sesiones_linea"},
		{ID: "item-035", CategoryCode: "9", ItemCode: "9.1.1", Description: "Foros - Nombre del Foro de Dudas e Inquietudes según lineamiento", MaxEvidences: 1, SortOrder: 35, GroupName: "foros"},
		{ID: "item-036", CategoryCode: "9", ItemCode: "9.1.2", Description: "Foros - Disponibilidad de mínimo un Foro de Dudas e Inquietudes", MaxEvidences: 1, SortOrder: 36, GroupName: "foros"},
		{ID: "item-037", CategoryCode: "9", ItemCode: "9.1.3", Description: "Foros - Configuración del Foro Temático con fecha de inicio y fin", MaxEvidences: 5, SortOrder: 37, GroupName: "foros"},
		{ID: "item-038", CategoryCode: "9", ItemCode: "9.1.4", Description: "Foros - Apertura del Foro Temático en las fechas establecidas", MaxEvidences: 5, SortOrder: 38, GroupName: "foros"},
		{ID: "item-039", CategoryCode: "9", ItemCode: "9.1.5", Description: "Foros - Respuesta a dudas en un plazo máximo de un día hábil", MaxEvidences: 1, SortOrder: 39, GroupName: "foros"},
		{ID: "item-040", CategoryCode: "9", ItemCode: "9.1.6", Description: "Foros - Instructor responde foros temáticos en un plazo de un día hábil", MaxEvidences: 5, SortOrder: 40, GroupName: "foros"},
		{ID: "item-041", CategoryCode: "9", ItemCode: "9.1.7", Description: "Foros - Retroalimentación de los Foros Temáticos disponibles", MaxEvidences: 5, SortOrder: 41, GroupName: "foros"},
		{ID: "item-042", CategoryCode: "10", ItemCode: "10.1.1", Description: "Evidencias - Retroalimentación y calificación de evidencias", MaxEvidences: 5, SortOrder: 42, GroupName: "evidencias_aprendizaje"},
		{ID: "item-043", CategoryCode: "10", ItemCode: "10.1.2", Description: "Evidencias - Retroalimentación en plazo máximo de tres días hábiles", MaxEvidences: 5, SortOrder: 43, GroupName: "evidencias_aprendizaje"},
		{ID: "item-044", CategoryCode: "11", ItemCode: "11.1.1", Description: "Anuncio de inicio de fase - Nombre de la fase del proyecto", MaxEvidences: 5, SortOrder: 44, GroupName: "anuncios_fase"},
		{ID: "item-045", CategoryCode: "11", ItemCode: "11.1.2", Description: "Anuncio de inicio de fase - Fecha de inicio y finalización de fase", MaxEvidences: 5, SortOrder: 45, GroupName: "anuncios_fase"},
		{ID: "item-046", CategoryCode: "11", ItemCode: "11.1.3", Description: "Anuncio de inicio de fase - Instrucciones sobre qué consultar", MaxEvidences: 5, SortOrder: 46, GroupName: "anuncios_fase"},
		{ID: "item-047", CategoryCode: "11", ItemCode: "11.1.4", Description: "Anuncio de inicio de fase - Pasos a seguir para el desarrollo", MaxEvidences: 5, SortOrder: 47, GroupName: "anuncios_fase"},
		{ID: "item-048", CategoryCode: "11", ItemCode: "11.2.1", Description: "Anuncios - Anuncio(s) de inicio de actividad de proyecto", MaxEvidences: 5, SortOrder: 48, GroupName: "anuncios_semanales"},
		{ID: "item-049", CategoryCode: "11", ItemCode: "11.2.2", Description: "Anuncios - Anuncio(s) de cierre de actividad de proyecto", MaxEvidences: 5, SortOrder: 49, GroupName: "anuncios_semanales"},
		{ID: "item-050", CategoryCode: "11", ItemCode: "11.2.3", Description: "Anuncios - Anuncio semanal de invitación a sesión en línea", MaxEvidences: 15, SortOrder: 50, GroupName: "anuncios_semanales"},
		{ID: "item-051", CategoryCode: "11", ItemCode: "11.3", Description: "Anuncio de aprendices aprobados de la fase publicado", MaxEvidences: 3, SortOrder: 51, GroupName: "anuncios_semanales"},
		{ID: "item-052", CategoryCode: "11", ItemCode: "11.4", Description: "Anuncios con función comunicativa y formato según orientaciones", MaxEvidences: 8, SortOrder: 52, GroupName: "anuncios_semanales"},
		{ID: "item-053", CategoryCode: "12", ItemCode: "12.1.1", Description: "Grabaciones semanales de sesiones en línea publicadas", MaxEvidences: 6, SortOrder: 53, GroupName: "sesiones_semanales"},
		{ID: "item-054", CategoryCode: "12", ItemCode: "12.1.2", Description: "Resumen de las sesiones semanales publicado en la subsección", MaxEvidences: 6, SortOrder: 54, GroupName: "sesiones_semanales"},
		{ID: "item-055", CategoryCode: "13", ItemCode: "13.1.1", Description: "Seguimiento - Subsección Reuniones EEF con 2 actas mensuales", MaxEvidences: 3, SortOrder: 55, GroupName: "documentos_retencion"},
		{ID: "item-056", CategoryCode: "13", ItemCode: "13.1.2", Description: "Seguimiento - Subsección Comités con mínimo un acta al terminar fase", MaxEvidences: 13, SortOrder: 56, GroupName: "documentos_retencion"},
		{ID: "item-057", CategoryCode: "13", ItemCode: "13.1.3", Description: "Seguimiento - Subsección Documentos de retención con un documento", MaxEvidences: 13, SortOrder: 57, GroupName: "documentos_retencion"},
		{ID: "item-058", CategoryCode: "13", ItemCode: "13.2.1", Description: "Reporte del Curso al final de fase - Copia de calificaciones al 100%", MaxEvidences: 13, SortOrder: 58, GroupName: "documentos_retencion"},
		{ID: "item-059", CategoryCode: "13", ItemCode: "13.2.2", Description: "Reporte del Curso al final de fase - Formatos de cierre", MaxEvidences: 13, SortOrder: 59, GroupName: "documentos_retencion"},
		{ID: "item-060", CategoryCode: "14", ItemCode: "14.1.1", Description: "Foros - Conclusión del Foro Temático en cronograma o día siguiente", MaxEvidences: 5, SortOrder: 60, GroupName: "conclusion_foros"},
		{ID: "item-061", CategoryCode: "14", ItemCode: "14.1.2", Description: "Foros - Conclusión del Foro Temático publicada como respuesta nueva", MaxEvidences: 5, SortOrder: 61, GroupName: "conclusion_foros"},
		{ID: "item-062", CategoryCode: "15", ItemCode: "15.1", Description: "Lenguaje cortés y respetuoso con uso de netiqueta y buena ortografía", MaxEvidences: 1, SortOrder: 62, GroupName: "netiqueta"},
	}
}

func ValidateDefinitions() error {
	items := Items()
	if len(items) != 62 {
		return fmt.Errorf("el catálogo de checklist debe tener 62 ítems, tiene %d", len(items))
	}
	seen := map[string]bool{}
	for _, item := range items {
		if item.ItemCode == "" || seen[item.ItemCode] || item.MaxEvidences < 1 {
			return fmt.Errorf("definición inválida para %q", item.ItemCode)
		}
		seen[item.ItemCode] = true
	}
	return nil
}
