package validate

const (
	PorterAnnotationKey   = "lb.kubesphere.io/v1apha1"
	PorterAnnotationValue = "porter"
)

func HasPorterLBAnnotation(annotation map[string]string) bool {
	if annotation == nil {
		return false
	}
	if value, ok := annotation[PorterAnnotationKey]; ok {
		if value == PorterAnnotationValue {
			return true
		}
	}
	return false
}
