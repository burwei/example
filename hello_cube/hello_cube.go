package main

import (
	"fmt"
	"go/build"
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	windowWidth  = 800
	windowHeight = 600
	vertexShader = `
		#version 330

		uniform mat4 projection;
		uniform mat4 camera;
		uniform mat4 model;

		in vec3 vert;
		in vec2 vertTexCoord;

		out vec2 fragTexCoord;

		void main() {
		fragTexCoord = vertTexCoord;
		gl_Position = projection * camera * model * vec4(vert, 1);
		}
	` + "\x00"
	fragmentShader = `
		#version 330

		uniform sampler2D tex;

		in vec2 fragTexCoord;

		out vec4 outputColor;

		void main() {
		outputColor = texture(tex, fragTexCoord);
		}
	` + "\x00"
)

var cubeVertices = []float32{
	//  X, Y, Z, U, V
	// Top (+Z)
	-1.0, -1.0, 1.0, 1.0, 0.0,
	1.0, -1.0, 1.0, 0.0, 0.0,
	-1.0, 1.0, 1.0, 1.0, 1.0,
	1.0, -1.0, 1.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 0.0, 1.0,
	-1.0, 1.0, 1.0, 1.0, 1.0,

	// Bottom (-Z)
	-1.0, -1.0, -1.0, 0.0, 0.0,
	-1.0, 1.0, -1.0, 0.0, 1.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	-1.0, 1.0, -1.0, 0.0, 1.0,
	1.0, 1.0, -1.0, 1.0, 1.0,

	// Back (-X)
	-1.0, -1.0, 1.0, 0.0, 1.0,
	-1.0, 1.0, -1.0, 1.0, 0.0,
	-1.0, -1.0, -1.0, 0.0, 0.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,
	-1.0, 1.0, 1.0, 1.0, 1.0,
	-1.0, 1.0, -1.0, 1.0, 0.0,

	// Front (+X)
	1.0, -1.0, 1.0, 1.0, 1.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 0.0, 0.0,
	1.0, -1.0, 1.0, 1.0, 1.0,
	1.0, 1.0, -1.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 0.0, 1.0,

	// Right (-Y)
	-1.0, -1.0, -1.0, 0.0, 0.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,
	1.0, -1.0, -1.0, 1.0, 0.0,
	1.0, -1.0, 1.0, 1.0, 1.0,
	-1.0, -1.0, 1.0, 0.0, 1.0,

	// Left (+Y)
	-1.0, 1.0, -1.0, 0.0, 0.0,
	-1.0, 1.0, 1.0, 0.0, 1.0,
	1.0, 1.0, -1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 1.0, 0.0,
	-1.0, 1.0, 1.0, 0.0, 1.0,
	1.0, 1.0, 1.0, 1.0, 1.0,
}

type simpleObj struct {
	program        uint32
	vao            uint32
	vertices       *[]float32
	model          mgl32.Mat4
	modelUniform   int32
	texture        uint32
	textureUniform int32
}

type simpleViewPoint struct {
	projection        mgl32.Mat4
	projectionUniform int32
	fovy              float32
	aspect            float32
	near              float32
	far               float32
	camera            mgl32.Mat4
	cameraUniform     int32
	eye               mgl32.Vec3
	center            mgl32.Vec3
	top               mgl32.Vec3
}

func main() {
	initThreadAndPath()
	window := initGlfwAndOpenGL(windowWidth, windowHeight)
	defer glfw.Terminate()

	vp := newViewPoint(windowWidth, windowHeight)

	cube1 := simpleObj{}
	newProgram(&cube1, vertexShader, fragmentShader)
	newMatrixes(&cube1, &vp)
	newTexture(&cube1, "square.png")
	newVao(&cube1, &cubeVertices)

	cube2 := simpleObj{}
	newProgram(&cube2, vertexShader, fragmentShader)
	newMatrixes(&cube2, &vp)
	newTexture(&cube2, "square.png")
	newVao(&cube2, &cubeVertices)

	initGlobalSettings()

	angle := 0.0
	previousTime := glfw.GetTime()

	for !window.ShouldClose() {
		// Clear before redraw
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// --- Drawing starts ---
		// update variables
		time := glfw.GetTime()
		elapsed := time - previousTime
		previousTime = time
		angle += elapsed
		cube1.model = mgl32.HomogRotate3D(float32(angle)/5, mgl32.Vec3{1, 0, 0})
		cube2.model = mgl32.HomogRotate3D(float32(angle)/2, mgl32.Vec3{1, 0, 0})

		// Render
		render(&cube1, &vp)
		render(&cube2, &vp)
		// --- Drawing ends ---

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func initThreadAndPath() {
	runtime.LockOSThread()
	dir, err := importPathToDir("github.com/go-gl/example/gl41core-cube")
	if err != nil {
		log.Fatalln("Unable to find Go package in your GOPATH, it's needed to load assets:", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		log.Panicln("os.Chdir:", err)
	}
}

func importPathToDir(importPath string) (string, error) {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		return "", err
	}
	return p.Dir, nil
}

func initGlfwAndOpenGL(width int, height int) *glfw.Window {
	// init GLFW
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(width, height, "Conway's Game of Life", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	// init OpenGL
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	return window
}

func initGlobalSettings() {
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(1.0, 1.0, 1.0, 1.0)
}

func newProgram(obj *simpleObj, vertexShaderSource, fragmentShaderSource string) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}
	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	gl.UseProgram(program)

	obj.program = program
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newViewPoint(width int, height int) simpleViewPoint {
	vp := simpleViewPoint{}
	vp.fovy = mgl32.DegToRad(45.0)
	vp.aspect = float32(width) / float32(height)
	vp.near = 0.1
	vp.far = 10
	vp.eye = mgl32.Vec3{3, 3, 3}
	vp.center = mgl32.Vec3{0, 0, 0}
	vp.top = mgl32.Vec3{0, 0, 1}
	return vp
}

func newMatrixes(obj *simpleObj, vp *simpleViewPoint) {
	vp.projection = mgl32.Perspective(vp.fovy, vp.aspect, vp.near, vp.far)
	vp.projectionUniform = gl.GetUniformLocation(obj.program, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(vp.projectionUniform, 1, false, &vp.projection[0])

	vp.camera = mgl32.LookAtV(vp.eye, vp.center, vp.top)
	vp.cameraUniform = gl.GetUniformLocation(obj.program, gl.Str("camera\x00"))
	gl.UniformMatrix4fv(vp.cameraUniform, 1, false, &vp.camera[0])

	obj.model = mgl32.Ident4()
	obj.modelUniform = gl.GetUniformLocation(obj.program, gl.Str("model\x00"))
	gl.UniformMatrix4fv(obj.modelUniform, 1, false, &obj.model[0])

	obj.textureUniform = gl.GetUniformLocation(obj.program, gl.Str("tex\x00"))
	gl.Uniform1i(obj.textureUniform, 0)

	gl.BindFragDataLocation(obj.program, 0, gl.Str("outputColor\x00"))
}

func newVao(obj *simpleObj, vertices *[]float32) {
	obj.vertices = vertices

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(*vertices)*4, gl.Ptr(*vertices), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(obj.program, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 3, gl.FLOAT, false, 5*4, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(obj.program, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, 5*4, 3*4)

	obj.vao = vao
}

func newTexture(obj *simpleObj, file string) {
	imgFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		panic(err)
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		panic(fmt.Errorf("unsupported stride"))
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	obj.texture = texture
}

func render(obj *simpleObj, vp *simpleViewPoint) {
	gl.UseProgram(obj.program)
	gl.UniformMatrix4fv(vp.projectionUniform, 1, false, &vp.projection[0])
	gl.UniformMatrix4fv(vp.cameraUniform, 1, false, &vp.camera[0])
	gl.UniformMatrix4fv(obj.modelUniform, 1, false, &obj.model[0])
	gl.BindVertexArray(obj.vao)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, obj.texture)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(*obj.vertices)/5)) // 5: X,Y,Z,U,V
}
