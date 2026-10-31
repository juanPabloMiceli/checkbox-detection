export interface DetectionSettings {
  outlineEnabled: boolean;
  lineEnabled: boolean;
  checkedInkRatio: number;
  minSidePx: number;
  maxSidePx: number;
  matchToleranceRatio: number;
  minRectangularity: number;
  finalIoUThreshold: number;
  minWidthPx: number;
  maxWidthPx: number;
  minEdgeSeparationPx: number;
  maxEdgeSeparationPx: number;
  minStrongSides: number;
  minCoverage: number;
}
export type DetectorKey = 'outlineEnabled' | 'lineEnabled';
export type SettingKey = Exclude<keyof DetectionSettings, DetectorKey>;
export interface SettingsDefinition {
  defaults: DetectionSettings;
  limits: Record<
    SettingKey,
    { min: number; max: number; integer: boolean; exclusiveMin?: boolean }
  >;
}
export interface SettingField {
  key: SettingKey;
  pairedKey?: SettingKey;
  name: string;
  help: string;
  unit: '%' | 'px' | 'sides';
}
export const SETTING_GROUPS: {
  name: string;
  description: string;
  detector?: DetectorKey;
  fields: SettingField[];
}[] = [
  {
    name: 'Shared · Checked marks',
    description: 'Classifies boxes found by either detector.',
    fields: [
      {
        key: 'checkedInkRatio',
        name: 'Checked mark threshold',
        unit: '%',
        help: 'Minimum ink coverage inside a checkbox to classify it as checked. Lower values recognize lighter or smaller marks; higher values require stronger marks.',
      },
    ],
  },
  {
    name: 'Detector 1 · Rectangular outlines',
    detector: 'outlineEnabled',
    description: 'Finds rectangular shapes. Each size limit applies to both width and height.',
    fields: [
      {
        key: 'minSidePx',
        name: 'Minimum width & height',
        unit: 'px',
        help: 'Outline detector only: both width and height must meet this minimum. A box excluded here may still be found by the border-line detector.',
      },
      {
        key: 'maxSidePx',
        name: 'Maximum width & height',
        unit: 'px',
        help: 'Outline detector only: neither width nor height may exceed this maximum. This does not change the border-line detector’s limits.',
      },
      {
        key: 'minRectangularity',
        name: 'Shape strictness',
        unit: '%',
        help: 'How closely an outline must fill its bounding rectangle. Higher values reject irregular shapes but may miss damaged checkboxes.',
      },
      {
        key: 'minStrongSides',
        name: 'Required visible sides',
        unit: 'sides',
        help: 'Number of sides that must show enough border ink. Requiring fewer sides can recover damaged boxes but may introduce false detections.',
      },
      {
        key: 'minCoverage',
        name: 'Outline border coverage',
        unit: '%',
        help: 'Minimum visible ink coverage for an outline side to count as present. Lower values tolerate more gaps.',
      },
    ],
  },
  {
    name: 'Detector 2 · Border lines',
    detector: 'lineEnabled',
    description:
      'Builds boxes from horizontal and vertical borders. Each size limit applies to both width and height.',
    fields: [
      {
        key: 'minWidthPx',
        pairedKey: 'minEdgeSeparationPx',
        name: 'Minimum width & height',
        unit: 'px',
        help: 'Border-line detector only: both width and height must meet this minimum. This does not change the outline detector’s limits.',
      },
      {
        key: 'maxWidthPx',
        pairedKey: 'maxEdgeSeparationPx',
        name: 'Maximum width & height',
        unit: 'px',
        help: 'Border-line detector only: neither width nor height may exceed this maximum. This does not change the outline detector’s limits.',
      },
    ],
  },
  {
    name: 'Shared · Matching & duplicates',
    description: 'Filters and combines results from both detectors.',
    fields: [
      {
        key: 'matchToleranceRatio',
        name: 'Checkbox size tolerance',
        unit: '%',
        help: 'Allowed variation from the checkbox sizes learned from the document. Higher values accept more size variation. Requires the outline detector to learn reference sizes.',
      },
      {
        key: 'finalIoUThreshold',
        name: 'Duplicate overlap threshold',
        unit: '%',
        help: 'Overlap required to treat two detections as the same checkbox. Lower values remove overlapping detections more aggressively.',
      },
    ],
  },
];
