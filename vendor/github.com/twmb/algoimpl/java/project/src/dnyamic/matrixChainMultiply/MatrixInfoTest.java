package dnyamic.matrixChainMultiply;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

public class MatrixInfoTest {

	@Test
	public void testChainMultiply() {
		// From CLRS ed. 3 page 376
		MatrixInfo[] matrices = new MatrixInfo[]{
				new MatrixInfo(30,35),
				new MatrixInfo(35,15),
				new MatrixInfo(15,5),
				new MatrixInfo(5,10),
				new MatrixInfo(10,20),
				new MatrixInfo(20,25)};
		assertEquals(15125, MatrixInfo.chainMultiply(matrices));
		
		matrices = new MatrixInfo[]{
				new MatrixInfo(2, 3),
				new MatrixInfo(3, 2),
		};
		assertEquals(12, MatrixInfo.chainMultiply(matrices));
		
		matrices = new MatrixInfo[]{
				new MatrixInfo(1,1),
		};
		assertEquals(0, MatrixInfo.chainMultiply(matrices));
		
		matrices = new MatrixInfo[]{};
		assertEquals(0, MatrixInfo.chainMultiply(matrices));
	}
}
