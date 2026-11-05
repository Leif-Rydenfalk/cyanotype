package model

type Quaternion [4]float64
type Vec3 [3]float64

func (q Quaternion) Mul(r Quaternion) Quaternion {
	return Quaternion{
		q[3]*r[0] + q[0]*r[3] + q[1]*r[2] - q[2]*r[1],
		q[3]*r[1] - q[0]*r[2] + q[1]*r[3] + q[2]*r[0],
		q[3]*r[2] + q[0]*r[1] - q[1]*r[0] + q[2]*r[3],
		q[3]*r[3] - q[0]*r[0] - q[1]*r[1] - q[2]*r[2],
	}
}

func (q Quaternion) Rotate(v Vec3) Vec3 {
	u := Vec3{q[0], q[1], q[2]}
	s := q[3]

	dot := u[0]*v[0] + u[1]*v[1] + u[2]*v[2]
	uu := u[0]*u[0] + u[1]*u[1] + u[2]*u[2]

	cross := Vec3{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	}

	return Vec3{
		2*dot*u[0] + (s*s-uu)*v[0] + 2*s*cross[0],
		2*dot*u[1] + (s*s-uu)*v[1] + 2*s*cross[1],
		2*dot*u[2] + (s*s-uu)*v[2] + 2*s*cross[2],
	}
}

func (q Quaternion) Inverse() Quaternion {
	n := q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3]
	return Quaternion{-q[0] / n, -q[1] / n, -q[2] / n, q[3] / n}
}

func (p *Placement) Inverse() *Placement {
	r := p.Rotation.Inverse()
	return &Placement{
		Rotation: r,
		Position: r.Rotate(Vec3{
			-p.Position[0],
			-p.Position[1],
			-p.Position[2],
		}),
	}
}

func (v Vec3) Add(v2 Vec3) Vec3 {
	return Vec3{
		v[0] + v2[0],
		v[1] + v2[1],
		v[2] + v2[2],
	}
}

type Placement struct {
	Rotation Quaternion `json:"rotation" yaml:"rotation"`
	Position Vec3       `json:"position" yaml:"position"`
}

var IdentityQuaternion = Quaternion{0, 0, 0, 1}
var IdentityVec3 = Vec3{0, 0, 0}

var IdentityPlacement = Placement{
	Rotation: IdentityQuaternion,
	Position: IdentityVec3,
}

// Compose returns the placement obtained by applying p first, then p2.
// For child->parent p and parent->grandparent p2, the result is child->grandparent.
func (p *Placement) Compose(p2 *Placement) *Placement {
	return &Placement{
		Rotation: p2.Rotation.Mul(p.Rotation),
		Position: p2.Rotation.Rotate(p.Position).Add(p2.Position),
	}
}

func (p *Placement) Align(p_ref *Placement, p2_ref *Placement) *Placement {
	return p2_ref.Inverse().Compose(p_ref).Compose(p)
}

func (p *Placement) Mate(p_ref *Placement, p2_ref *Placement) *Placement {
	flip := &Placement{
		Rotation: Quaternion{1, 0, 0, 0},
		Position: IdentityVec3,
	}
	return p2_ref.Inverse().Compose(flip).Compose(p_ref).Compose(p)
}
