package mater

import (
	"mater/vect"
	"gl"
)

type Mater struct {
	DefaultCamera Camera
	ScreenSize vect.Vect
	Running, Paused bool
	Dbg DebugData
	Scene *Scene
	OnKeyCallback OnKeyCallbackFunc
}

func (mater *Mater) Init () {
	dbg := &(mater.Dbg)
	dbg.Init(mater)
	mater.Scene = new(Scene)
	mater.Scene.Init(mater)

	if dbg.DebugView == nil {
		mater.Dbg.DebugView = NewDebugView(mater.Scene.Space)
	} else {
		mater.Dbg.DebugView.Reset(mater.Scene.Space)
	}

	mater.OnKeyCallback = DefaultKeyCallback
}

func (mater *Mater) OnResize (width, height int) {
	if height == 0 {
		height = 1
	}

	w, h := float64(width), float64(height)
	mater.ScreenSize = vect.Vect{w, h}
	mater.DefaultCamera.ScreenSize = mater.ScreenSize
	if mater.Scene != nil {
		mater.Scene.Camera.ScreenSize = mater.ScreenSize
	}
	
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	//camera centered at (0,0)
	gl.Ortho(0, w, h, 0, 1, -1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.Translated(.375, .375, 0)
}

func (mater *Mater) Update (dt float64) {
	
	mater.Scene.Update(dt)
}

func (mater *Mater) Draw () {
	gl.Clear(gl.COLOR_BUFFER_BIT)
	mater.Scene.Camera.PreDraw()
	{

	}
	mater.Scene.Camera.PostDraw()
}
