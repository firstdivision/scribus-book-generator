package imageplacement

import "fmt"

type Edge string

type PhysicalEdge string

type SnapTarget string

const (
	EdgeOutside Edge = "outside"
	EdgeInside  Edge = "inside"
	EdgeTop     Edge = "top"
	EdgeBottom  Edge = "bottom"
)

const (
	PhysicalLeft   PhysicalEdge = "left"
	PhysicalRight  PhysicalEdge = "right"
	PhysicalTop    PhysicalEdge = "top"
	PhysicalBottom PhysicalEdge = "bottom"
)

const (
	SnapTargetContentArea SnapTarget = "content_area"
	SnapTargetTrim        SnapTarget = "trim"
	SnapTargetBleed       SnapTarget = "bleed"
)

type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

type Spacing struct {
	Top     float64
	Bottom  float64
	Inside  float64
	Outside float64
}

type ResolvedSpacing struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

func IsRightPage(layoutMode, firstPageMode string, pageNumber int) bool {
	if layoutMode != "facing_pages" {
		return true
	}
	firstPageIsRight := firstPageMode == "right"
	if firstPageIsRight {
		return pageNumber%2 == 1
	}
	return pageNumber%2 == 0
}

func ResolveSemanticEdge(edge Edge, isRightPage bool) (PhysicalEdge, error) {
	switch edge {
	case EdgeOutside:
		if isRightPage {
			return PhysicalRight, nil
		}
		return PhysicalLeft, nil
	case EdgeInside:
		if isRightPage {
			return PhysicalLeft, nil
		}
		return PhysicalRight, nil
	case EdgeTop:
		return PhysicalTop, nil
	case EdgeBottom:
		return PhysicalBottom, nil
	default:
		return "", fmt.Errorf("unsupported semantic edge %q", edge)
	}
}

func ResolveWrapSpacing(spacing Spacing, isRightPage bool) ResolvedSpacing {
	if isRightPage {
		return ResolvedSpacing{
			Left:   spacing.Inside,
			Right:  spacing.Outside,
			Top:    spacing.Top,
			Bottom: spacing.Bottom,
		}
	}
	return ResolvedSpacing{
		Left:   spacing.Outside,
		Right:  spacing.Inside,
		Top:    spacing.Top,
		Bottom: spacing.Bottom,
	}
}

func ChooseEdge(explicit *Edge, allowed []Edge, preferred []Edge) (Edge, error) {
	if len(allowed) == 0 {
		return "", fmt.Errorf("no allowed edges configured")
	}

	allowedSet := map[Edge]bool{}
	for _, edge := range allowed {
		if !IsValidEdge(edge) {
			return "", fmt.Errorf("unsupported allowed edge %q", edge)
		}
		allowedSet[edge] = true
	}

	if explicit != nil {
		if !IsValidEdge(*explicit) {
			return "", fmt.Errorf("unsupported explicit edge %q", *explicit)
		}
		if !allowedSet[*explicit] {
			return "", fmt.Errorf("explicit edge %q is not allowed", *explicit)
		}
		return *explicit, nil
	}

	for _, edge := range preferred {
		if !IsValidEdge(edge) {
			return "", fmt.Errorf("unsupported preferred edge %q", edge)
		}
		if allowedSet[edge] {
			return edge, nil
		}
	}

	return allowed[0], nil
}

func SnapFrame(target Rect, frameWidth, frameHeight float64, semanticEdge Edge, isRightPage bool, edgeGap float64) (float64, float64, PhysicalEdge, error) {
	if edgeGap < 0 {
		return 0, 0, "", fmt.Errorf("edge gap must be >= 0")
	}
	if frameWidth <= 0 || frameHeight <= 0 {
		return 0, 0, "", fmt.Errorf("frame dimensions must be > 0")
	}
	if target.Width <= 0 || target.Height <= 0 {
		return 0, 0, "", fmt.Errorf("target dimensions must be > 0")
	}

	physicalEdge, err := ResolveSemanticEdge(semanticEdge, isRightPage)
	if err != nil {
		return 0, 0, "", err
	}

	x := target.X
	y := target.Y

	switch physicalEdge {
	case PhysicalLeft:
		x = target.X + edgeGap
		y = target.Y + edgeGap
	case PhysicalRight:
		x = target.X + target.Width - frameWidth - edgeGap
		y = target.Y + edgeGap
	case PhysicalTop:
		y = target.Y + edgeGap
		if isRightPage {
			x = target.X + target.Width - frameWidth
		} else {
			x = target.X
		}
	case PhysicalBottom:
		y = target.Y + target.Height - frameHeight - edgeGap
		if isRightPage {
			x = target.X + target.Width - frameWidth
		} else {
			x = target.X
		}
	default:
		return 0, 0, "", fmt.Errorf("unsupported physical edge %q", physicalEdge)
	}

	return x, y, physicalEdge, nil
}

func ResolveSnapTargetRect(target SnapTarget, contentArea Rect, trim Rect, bleed Rect) (Rect, error) {
	switch target {
	case SnapTargetContentArea:
		return contentArea, nil
	case SnapTargetTrim:
		return trim, nil
	case SnapTargetBleed:
		return bleed, nil
	default:
		return Rect{}, fmt.Errorf("unsupported snap_target %q", target)
	}
}

func IsValidEdge(edge Edge) bool {
	switch edge {
	case EdgeOutside, EdgeInside, EdgeTop, EdgeBottom:
		return true
	default:
		return false
	}
}
