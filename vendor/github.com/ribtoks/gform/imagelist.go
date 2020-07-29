package gform

import (
	"github.com/ribtoks/w32"
)

type ImageList struct {
	handle w32.HIMAGELIST
}

func NewImageList(cx, cy int32, flags uint32, cInitial, cGrow int32) *ImageList {
	imgl := new(ImageList)
	imgl.handle = w32.ImageList_Create(cx, cy, flags, cInitial, cGrow)

	return imgl
}

func (this *ImageList) Handle() w32.HIMAGELIST {
	return this.handle
}

func (this *ImageList) Destroy() bool {
	return w32.ImageList_Destroy(this.handle)
}

func (this *ImageList) SetImageCount(uNewCount uint32) bool {
	return w32.ImageList_SetImageCount(this.handle, uNewCount)
}

func (this *ImageList) ImageCount() int32 {
	return w32.ImageList_GetImageCount(this.handle)
}

func (this *ImageList) AddIcon(icon *Icon) int32 {
	return w32.ImageList_AddIcon(this.handle, icon.Handle())
}

func (this *ImageList) RemoveAll() bool {
	return w32.ImageList_RemoveAll(this.handle)
}

func (this *ImageList) Remove(i int32) bool {
	return w32.ImageList_Remove(this.handle, i)
}
