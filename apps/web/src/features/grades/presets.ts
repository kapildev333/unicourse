import type { GradeScaleRow } from './queries'

export const usPreset: GradeScaleRow[] = [
  { letter: 'A', minPercent: 93, points: 4 }, { letter: 'A-', minPercent: 90, points: 3.7 }, { letter: 'B+', minPercent: 87, points: 3.3 },
  { letter: 'B', minPercent: 83, points: 3 }, { letter: 'B-', minPercent: 80, points: 2.7 }, { letter: 'C+', minPercent: 77, points: 2.3 },
  { letter: 'C', minPercent: 73, points: 2 }, { letter: 'C-', minPercent: 70, points: 1.7 }, { letter: 'D+', minPercent: 67, points: 1.3 },
  { letter: 'D', minPercent: 63, points: 1 }, { letter: 'D-', minPercent: 60, points: 0.7 }, { letter: 'F', minPercent: 0, points: 0 },
]
export const canadianPreset: GradeScaleRow[] = [
  { letter: 'A+', minPercent: 90, points: 4 }, { letter: 'A', minPercent: 85, points: 4 }, { letter: 'A-', minPercent: 80, points: 3.7 },
  { letter: 'B+', minPercent: 77, points: 3.3 }, { letter: 'B', minPercent: 73, points: 3 }, { letter: 'B-', minPercent: 70, points: 2.7 },
  { letter: 'C+', minPercent: 67, points: 2.3 }, { letter: 'C', minPercent: 63, points: 2 }, { letter: 'C-', minPercent: 60, points: 1.7 },
  { letter: 'D+', minPercent: 57, points: 1.3 }, { letter: 'D', minPercent: 53, points: 1 }, { letter: 'D-', minPercent: 50, points: 0.7 },
  { letter: 'F', minPercent: 0, points: 0 },
]
