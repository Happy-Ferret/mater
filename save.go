package mater

import (
	"bytes"
	"json"
	"os"
	"box2d"
)

var saveDirectory = "saves/"

func (mater *Mater) SaveScene (path string) os.Error{
	scene := mater.Scene

	path = saveDirectory + path

	file, err := os.Create(path)
	if err != nil {
		dbg.Printf("Error opening File: %v", err)
		return err
	}
	defer file.Close()

	//encoder := json.NewEncoder(file)
	//err = encoder.Encode(scene)

	dataString, err := json.MarshalIndent(scene, "", "\t")
	if err != nil {
		dbg.Printf("Error encoding Scene: %v", err)
		return err
	}

	buf := bytes.NewBuffer(dataString)
	n, err := buf.WriteTo(file)
	if err != nil {
		dbg.Printf("Error after writing %v characters to File: %v", n, err)
		return err
	}

	return nil
}

func (mater *Mater) LoadScene (path string) os.Error {

	var scene *Scene

	path = saveDirectory + path

	file, err := os.Open(path)
	if err != nil {
		dbg.Printf("Error opening File: %v", err)
		return err
	}
	defer file.Close()

	scene = new(Scene)
	decoder := json.NewDecoder(file)

	err = decoder.Decode(scene)
	if err != nil {
		dbg.Printf("Error decoding Scene: %v", err)
		return err
	}

	mater.Scene = scene
	scene.World.Enabled = true

	if mater.Scene.Camera == nil {
		cam := mater.DefaultCamera
		mater.Scene.Camera = &cam
	} else {
		mater.Scene.Camera.ScreenSize = mater.ScreenSize
	}

	mater.Dbg.DebugView.Reset(mater.Scene.World)

	return nil
}

type serializationState struct {
	SerializedBodies map[*box2d.Body]bool
}

func (scene *Scene) MarshalJSON() ([]byte, os.Error) {
	bodyNum := len(scene.World.BodyList())
	state := serializationState{
		//allocate space for half of the bodies
		//not all are going to be attached to entities
		SerializedBodies: make(map[*box2d.Body]bool, bodyNum / 2),
	}

	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)

	var err os.Error

	buf.WriteString(`{"Camera":`)
	encoder.Encode(scene.Camera)

	buf.WriteString(`,"Entities":`)
	entities, err := scene.MarshalEntities(&state)
	if err != nil {
		return nil, err
	}
	buf.Write(entities)


	buf.WriteString(`,"World":`)
	world, err := scene.MarshalWorld(&state)
	if err != nil {
		return nil, err
	}
	buf.Write(world)

	buf.WriteByte('}')

	return buf.Bytes(), nil
}

func (scene *Scene) MarshalEntities(state *serializationState) ([]byte, os.Error) {
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)

	buf.WriteByte('[')
	for _, entity := range scene.Entities {
		body := entity.Body
		if body != nil {
			state.SerializedBodies[body] = true
		}
		//two entities can not be attached to the same body.
		//if they are, once unserialized each of them has its own copy.

		//workdaround till update: set Entity.Scene to nil so we don't serialize it
		entity.Scene = nil

		err := encoder.Encode(entity)
		if err != nil {
			return nil, err
		}

		buf.WriteByte(',')

		//restore Entity.Scene
		entity.Scene = scene	
	}
	if len(scene.Entities) > 0 {
		//cut trailing comma
		buf.Truncate(buf.Len() - 1)
	}

	buf.WriteByte(']')

	return buf.Bytes(), nil
}

func (scene *Scene) MarshalWorld(state *serializationState) ([]byte, os.Error) {
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)

	world := scene.World

	buf.WriteByte('{')
	buf.WriteString(`"Gravity":`)
	encoder.Encode(world.Gravity)

	buf.WriteString(`,"Bodies":`)
	buf.WriteByte('[')

	//actual number of serialized bodies may be different than the total number of bodies
	//because of that we keep track how many we actually write
	bodyNum := 0
	for _, body := range world.BodyList() {
		
		if state.SerializedBodies[body] {
			continue
		}
		bodyNum++

		err := encoder.Encode(body)
		if err != nil {
			return nil, err
		}

		buf.WriteByte(',')
	}

	if bodyNum > 0 {
		//cut trailing comma
		buf.Truncate(buf.Len() - 1)
	}
	buf.WriteByte(']')

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (scene *Scene) UnmarshalJSON(data []byte) os.Error {
	sceneData := struct {
		Camera *Camera
		World *box2d.World
		Entities []json.RawMessage
	}{}

	err := json.Unmarshal(data, &sceneData)
	if err != nil {
		return err
	}

	sd := &sceneData

	scene.Camera = sd.Camera
	scene.World = sd.World

	if scene.Entities == nil {
		scene.Entities = make([]*Entity, 0, 32)
	}

	for _, rawEntity := range sd.Entities {
		err := scene.UnmarshalEntity(rawEntity)
		if err != nil {
			return err
		}
	}

	return nil
}

func (scene *Scene) UnmarshalEntity(entityData []byte) os.Error {
	entity := Entity{}

	err := json.Unmarshal(entityData, &entity)
	if err != nil {
		return err
	}

	entity.Scene = scene
	if entity.Body != nil {
		entity.Body.RegisterBody(scene.World)
	}

	scene.Entities = append(scene.Entities, &entity)

	return nil
}

func (entity *Entity) MarshalJSON() ([]byte, os.Error) {
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)

	buf.WriteByte('{')
	buf.WriteString(`"ID":`)
	encoder.Encode(entity.id)

	buf.WriteString(`,"Enabled":`)
	encoder.Encode(entity.Enabled)

	buf.WriteString(`,"Body":`)
	err := encoder.Encode(entity.Body)
	if err != nil {
		return nil, err
	}

	buf.WriteString(`,"Components":`)
	buf.WriteByte('[')
	ccount := 0
	for _, component := range entity.Components {
		if sc, ok := serializableComponents[component.Name()]; ok {
			ccount++
			buf.WriteByte('"')
			buf.WriteString(component.Name())
			buf.WriteString(`":`)
			data, err := sc.MarshalJSON(entity)
			if err != nil {
				return nil, err
			}
			buf.Write(data)
			buf.WriteByte(',')
		}
	}

	if ccount > 0 {
		//cut trailing comma
		buf.Truncate(buf.Len() - 1)
	}

	buf.WriteByte(']')
	buf.WriteByte('}')

	return buf.Bytes(), nil
}