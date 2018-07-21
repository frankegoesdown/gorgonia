package main

import "text/template"

const binopRaw = `// {{.Method}} implements tensor.{{.Method}}er. It does not support safe or increment operation options and will return an error if those options are passed in
func (e *Engine) {{.Method}}(a tensor.Tensor, b tensor.Tensor, opts ...tensor.FuncOpt) (retVal tensor.Tensor, err error) {
	name := constructName2(a, b, "add")

	if !e.HasFunc(name) {
		return nil, errors.Errorf("Unable to perform Add(). The tensor engine does not have the function %q", name)
	}

	if err = binaryCheck(a, b); err != nil {
		return nil, errors.Wrap(err, "Basic checks failed for {{.Method}}")
	}

	var reuse tensor.DenseTensor
	var safe, toReuse bool
	if reuse, safe, toReuse, _, _, err = handleFuncOpts(a.Shape(), a.Dtype(), a.DataOrder(), true, opts...); err != nil {
		return nil, errors.Wrap(err, "Unable to handle funcOpts")
	}

	var mem, memB cu.DevicePtr
	var size int64

	switch {
	case toReuse:
		mem = cu.DevicePtr(reuse.Uintptr())
		memA := cu.DevicePtr(a.Uintptr())
		memSize := int64(a.MemSize())
		e.Memcpy(mem, memA)

		size = int64(logicalSize(reuse.Shape()))
		retVal = reuse
	case !safe:
		mem = cu.DevicePtr(a.Uintptr())
		retVal = reuse
		size = int64(logicalSize(a.Shape()))
	default:
		return nil, errors.New("Impossible state: A reuse tensor must be passed in, or the operation must be unsafe. Incr and safe operations are not supported")
	}

	fn := e.f[name]
	gridDimX, gridDimY, gridDimZ, blockDimX, blockDimY, blockDimZ := e.ElemGridSize(int(size))
	args := []unsafe.Pointer{
		unsafe.Pointer(&mem),
		unsafe.Pointer(&memB),
		unsafe.Pointer(&size),
	}
	e.c.LaunchAndSync(fn, gridDimX, gridDimY, gridDimZ, blockDimX, blockDimY, blockDimZ, 0, cu.NoStream, args)
	return
}

// {{.ScalarMethod}}Scalar implements tensor.{{.Method}}er. It does not support safe or increment operation options and will return an error if those options are passed in
func (e *Engine) {{.ScalarMethod}}Scalar(a tensor.Tensor, b interface{}, leftTensor bool, opts ...tensor.FuncOpt) (retVal tensor.Tensor, err error) {
	name := constructName1(a, leftTensor, "add")
	if !e.HasFunc(name) {
		return nil, errors.Errorf("Unable to perform Add(). The tensor engine does not have the function %q", name)
	}

	if err = unaryCheck(a); err != nil {
		return nil, errors.Wrap(err, "Basic checks failed for {{.ScalarMethod}}Scalar")
	}

	var reuse tensor.DenseTensor
	var safe, toReuse bool
	if reuse, safe, toReuse, _, _, err = handleFuncOpts(a.Shape(), a.Dtype(), a.DataOrder(), true, opts...); err != nil {
		return nil, errors.Wrap(err, "Unable to handle funcOpts")
	}

	var mem, memB cu.DevicePtr
	var size int64

	switch {
	case toReuse:
		mem = cu.DevicePtr(reuse.Uintptr())
		memA := cu.DevicePtr(a.Uintptr())
		memSize := int64(a.MemSize())
		e.Memcpy(mem, memA)

		size = int64(logicalSize(reuse.Shape()))
		retVal = reuse
	case !safe:
		mem = cu.DevicePtr(a.Uintptr())
		retVal = reuse
		size = int64(logicalSize(a.Shape()))
	default:
		return nil, errors.New("Impossible state: A reuse tensor must be passed in, or the operation must be unsafe. Incr and safe operations are not supported")
	}

	fn := e.f[name]
	gridDimX, gridDimY, gridDimZ, blockDimX, blockDimY, blockDimZ := e.ElemGridSize(int(size))
	args := []unsafe.Pointer{
		unsafe.Pointer(&mem),
		unsafe.Pointer(&memB),
		unsafe.Pointer(&size),
	}
	e.c.LaunchAndSync(fn, gridDimX, gridDimY, gridDimZ, blockDimX, blockDimY, blockDimZ, 0, cu.NoStream, args)
	return
}
`

var (
	binopTmpl *template.Template
)

func init() {
	binopTmpl = template.Must(template.New("binop").Funcs(funcmap).Parse(binopRaw))
}
