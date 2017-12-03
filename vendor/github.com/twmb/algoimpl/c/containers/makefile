ALL_TEST_SRC 	:= $(shell find . -name '*_testspecial.c')
ALL_TEST_EXE	:= ${ALL_TEST_SRC:.c=}

CFLAGS := -g -Wall -pedantic -std=c11

special: heap_testspecial dynamic_array_testspecial

test:	special
	@for test in ${ALL_TEST_EXE} ;            \
	  do                                      \
	    echo testing $${test} ;               \
	    ./$${test} ;                          \
	    if [ $$? -ne 0 ] ;                    \
	    then                                  \
	      /bin/echo -e "\e[01;31mfailed\e[m" ;\
	    else                                  \
	      echo passed ;                       \
	    fi ;                                  \
	    echo ;                                \
	  done 

heap_testspecial: heap_testspecial.c heap.c dynamic_array.c heap.h dynamic_array.h
	clang ${CFLAGS} -o $@ heap_testspecial.c dynamic_array.c -lpthread

dynamic_array_testspecial: dynamic_array_testspecial.c dynamic_array.c dynamic_array.h
	clang ${CFLAGS} -o $@ dynamic_array_testspecial.c -lpthread

clean:
	rm -f heap_testspecial dynamic_array_testspecial
