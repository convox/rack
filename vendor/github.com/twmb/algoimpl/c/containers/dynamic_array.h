#ifndef TWMB_DYNARRAY
#define TWMB_DYNARRAY

typedef struct dynamic_array dynarr;

// Creates and returns a new dynamic array.
// destroy_dynarr must be called to delete it.
dynarr *create_dynarr(void);
// Makes and returns a new dynamic array with the given length
// and capacity. destroy_dynarr must be called to delete it.
dynarr *make_dynarr(int len, int cap);
// Destroys a dynamic array. If there are no remaining slices
// of a dynamic array's elements, this destroys the elements
// the dynamic array contained.
void destroy_dynarr(dynarr *array);
// Appends to the end of a dynamic array. It reallocates
// elements and updates the capacity as necessary.
void *dynarr_append(dynarr *array, void *element);
// Slices and returns a new dynamic array. The return value must
// not overwrite the array being slice as that array must be destroyed
// for all memory to be freed.
dynarr *dynarr_slice(dynarr *array, int from, int to);
// Returns the element at the requested position in the array.
void *dynarr_at(dynarr *array, int position);
// Sets the element at the given position in the array to the new element.
void dynarr_set(dynarr *array, int position, void *element);
// Returns the current length of the dynamic array.
int dynarr_len(dynarr *array);

#endif
