export type BoundingBox = [x1: number, y1: number, x2: number, y2: number];

export interface Detection {
  bbox: BoundingBox;
  is_checked: boolean;
}

export interface DetectionResponse {
  boxes: Detection[];
}
