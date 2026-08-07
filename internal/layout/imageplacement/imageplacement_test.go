package imageplacement

import "testing"

func TestResolveSemanticEdgeFacingPages(t *testing.T) {
	leftOutside, err := ResolveSemanticEdge(EdgeOutside, false)
	if err != nil {
		t.Fatalf("ResolveSemanticEdge returned error: %v", err)
	}
	if leftOutside != PhysicalLeft {
		t.Fatalf("expected left page outside -> left, got %q", leftOutside)
	}

	leftInside, err := ResolveSemanticEdge(EdgeInside, false)
	if err != nil {
		t.Fatalf("ResolveSemanticEdge returned error: %v", err)
	}
	if leftInside != PhysicalRight {
		t.Fatalf("expected left page inside -> right, got %q", leftInside)
	}

	rightOutside, err := ResolveSemanticEdge(EdgeOutside, true)
	if err != nil {
		t.Fatalf("ResolveSemanticEdge returned error: %v", err)
	}
	if rightOutside != PhysicalRight {
		t.Fatalf("expected right page outside -> right, got %q", rightOutside)
	}

	rightInside, err := ResolveSemanticEdge(EdgeInside, true)
	if err != nil {
		t.Fatalf("ResolveSemanticEdge returned error: %v", err)
	}
	if rightInside != PhysicalLeft {
		t.Fatalf("expected right page inside -> left, got %q", rightInside)
	}
}

func TestChooseEdgePrefersOutside(t *testing.T) {
	edge, err := ChooseEdge(nil, []Edge{EdgeOutside, EdgeInside, EdgeTop, EdgeBottom}, []Edge{EdgeOutside, EdgeTop})
	if err != nil {
		t.Fatalf("ChooseEdge returned error: %v", err)
	}
	if edge != EdgeOutside {
		t.Fatalf("expected outside edge, got %q", edge)
	}
}

func TestSnapFrameAppliesEdgeGap(t *testing.T) {
	target := Rect{X: 10, Y: 20, Width: 100, Height: 80}
	x, y, physical, err := SnapFrame(target, 40, 30, EdgeOutside, false, 3)
	if err != nil {
		t.Fatalf("SnapFrame returned error: %v", err)
	}
	if physical != PhysicalLeft {
		t.Fatalf("expected physical left edge, got %q", physical)
	}
	if x != 13 {
		t.Fatalf("expected x=13, got %.4f", x)
	}
	if y != 23 {
		t.Fatalf("expected y=23, got %.4f", y)
	}
}

func TestResolveWrapSpacingFacingPages(t *testing.T) {
	spacing := Spacing{Top: 5, Bottom: 6, Inside: 7, Outside: 8}

	right := ResolveWrapSpacing(spacing, true)
	if right.Left != 7 || right.Right != 8 || right.Top != 5 || right.Bottom != 6 {
		t.Fatalf("unexpected right-page spacing %+v", right)
	}

	left := ResolveWrapSpacing(spacing, false)
	if left.Left != 8 || left.Right != 7 || left.Top != 5 || left.Bottom != 6 {
		t.Fatalf("unexpected left-page spacing %+v", left)
	}
}

func TestResolveSnapTargetRect(t *testing.T) {
	content := Rect{X: 10, Y: 20, Width: 30, Height: 40}
	trim := Rect{X: 0, Y: 0, Width: 100, Height: 80}
	bleed := Rect{X: -3, Y: -3, Width: 106, Height: 86}

	got, err := ResolveSnapTargetRect(SnapTargetContentArea, content, trim, bleed)
	if err != nil {
		t.Fatalf("ResolveSnapTargetRect returned error: %v", err)
	}
	if got != content {
		t.Fatalf("expected content area rect, got %+v", got)
	}
}
