package dnyamic.matrixChainMultiply;

public class MatrixInfo {
	public int rows, cols;
	
	public MatrixInfo(int rows, int cols) {
		this.rows = rows;
		this.cols = cols;
	}
	
	/** Finds the minimum number of multiply operations needed to multiply a 
	 *  chain sequence of matrices (each of which is represented by the MatrixInfo object).
	 *  TODO: add multiplication order (currently only number of multiplications)
	 * @param matrices An array of MatrixInfo objects
	 * @return the minimum number of operations needed to multiply the matrices.
	 */
	public static int chainMultiply(MatrixInfo[] matrices) {
		if (matrices.length <= 0) {
			return 0;
		}
		int[][] m = new int[matrices.length][matrices.length];
//		int[][] s = new int[matrices.length][matrices.length];
		for (int i = 0; i < matrices.length; i++) {
			m[i][i] = 0;
		}
		for (int l = 2; l <= matrices.length; l++) {
			for (int i = 0; i <= matrices.length - l; i++) {
				int j = i + l - 1;
				m[i][j] = Integer.MAX_VALUE;
				for (int k = i; k < j; k++) {
					int q = m[i][k] + m[k+1][j] + matrices[i].rows * matrices[k].cols * matrices[j].cols;
					if (q < m[i][j]) {
						m[i][j] = q;
//						s[i][j] = k;
					}
				}
			}
		}
		return m[0][matrices.length - 1];	
	}

}
