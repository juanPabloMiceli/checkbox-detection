import { afterNextRender, computed, inject, Injectable, signal } from '@angular/core';
import { finalize } from 'rxjs';
import { DetectionService } from './detection.service';
import {
  DetectionSettings,
  DetectorKey,
  SettingsDefinition,
  SettingKey,
  SETTING_GROUPS,
} from '../models/detection-settings.model';

@Injectable({ providedIn: 'root' })
export class DetectionSettingsStore {
  private readonly api = inject(DetectionService);
  readonly definition = signal<SettingsDefinition | null>(null);
  readonly draft = signal<Partial<Record<SettingKey, number | null>>>({});
  readonly enabled = signal<Record<DetectorKey, boolean>>({
    outlineEnabled: true,
    lineEnabled: true,
  });
  readonly detectorError = computed(() =>
    !this.enabled().outlineEnabled && !this.enabled().lineEnabled
      ? 'Enable at least one detector.'
      : null,
  );
  readonly loading = signal(false);
  readonly loadError = signal<string | null>(null);
  readonly fields = SETTING_GROUPS.flatMap((group) => group.fields);
  readonly errors = computed(() => {
    const definition = this.definition();
    const errors: Partial<Record<SettingKey, string>> = {};
    if (!definition) return errors;
    for (const field of this.fields) {
      if (!this.isFieldEnabled(field.key)) continue;
      const value = this.draft()[field.key];
      const limit = definition.limits[field.key];
      const factor = field.unit === '%' ? 100 : 1;
      if (value === null || value === undefined || !Number.isFinite(value)) {
        errors[field.key] = 'Enter a number.';
      } else if (limit.integer && !Number.isInteger(value)) {
        errors[field.key] = 'Enter a whole number.';
      } else if (limit.exclusiveMin && value === limit.min) {
        errors[field.key] = `Use a value greater than ${limit.min * factor}.`;
      } else if (value < limit.min || value > limit.max) {
        errors[field.key] = `Use a value between ${limit.min * factor} and ${limit.max * factor}.`;
      }
    }
    for (const [minimum, maximum] of [
      ['minSidePx', 'maxSidePx'],
      ['minWidthPx', 'maxWidthPx'],
    ] as const) {
      if (!this.isFieldEnabled(minimum)) continue;
      const min = this.draft()[minimum],
        max = this.draft()[maximum];
      if (typeof min === 'number' && typeof max === 'number' && min > max) {
        errors[minimum] = 'Minimum must not exceed maximum.';
        errors[maximum] = 'Maximum must not be below minimum.';
      }
    }
    return errors;
  });
  readonly valid = computed(
    () =>
      this.definition() !== null &&
      !this.loading() &&
      !this.detectorError() &&
      Object.keys(this.errors()).length === 0,
  );

  constructor() {
    afterNextRender(() => this.load());
  }

  load() {
    if (this.loading()) return;
    this.loading.set(true);
    this.loadError.set(null);
    this.api
      .getSettings()
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (definition) => {
          this.definition.set(definition);
          this.applyDefaults(definition.defaults);
        },
        error: () => {
          this.loadError.set('Could not load detection settings. Retry to enable processing.');
        },
      });
  }

  update(key: SettingKey, value: number | null) {
    this.draft.update((current) => ({ ...current, [key]: value }));
  }
  reset() {
    const defaults = this.definition()?.defaults;
    if (defaults) this.applyDefaults(defaults);
  }
  setDetector(key: DetectorKey, enabled: boolean) {
    this.enabled.update((current) => ({ ...current, [key]: enabled }));
  }

  isFieldEnabled(key: SettingKey): boolean {
    if (key === 'matchToleranceRatio' && !this.enabled().outlineEnabled) return false;
    const detector = SETTING_GROUPS.find((group) =>
      group.fields.some((field) => field.key === key),
    )?.detector;
    return !detector || this.enabled()[detector];
  }

  private applyDefaults(defaults: DetectionSettings) {
    const { outlineEnabled, lineEnabled, ...values } = defaults;
    this.enabled.set({ outlineEnabled, lineEnabled });
    this.draft.set(values);
  }

  snapshot(): DetectionSettings | null {
    const defaults = this.definition()?.defaults;
    if (!defaults || !this.valid()) return null;
    const settings = { ...defaults, ...this.enabled() };
    for (const field of this.fields) {
      if (!this.isFieldEnabled(field.key)) continue;
      const value = this.draft()[field.key] as number;
      settings[field.key] = value;
      if (field.pairedKey) settings[field.pairedKey] = value;
    }
    return settings;
  }
}
