import { Component, computed, input } from '@angular/core';
import { DetectionResponse } from '../models/detection.model';

interface JsonToken {
  text: string;
  kind: 'key' | 'string' | 'number' | 'boolean' | 'punctuation';
}

@Component({
  selector: 'app-json-response',
  standalone: true,
  templateUrl: './json-response.html',
  styleUrl: './json-response.scss',
})
export class JsonResponse {
  readonly response = input.required<DetectionResponse>();

  readonly lines = computed(() =>
    JSON.stringify(this.response(), null, 2)
      .split('\n')
      .map((line) => {
        const tokens: JsonToken[] = [];
        // Render text through interpolation, never injected HTML.
        const pattern =
          /"(?:\\.|[^"\\])*"|\b(?:true|false|null)\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|[^"\w\d-]+/g;
        for (const match of line.matchAll(pattern)) {
          const text = match[0];
          let kind: JsonToken['kind'] = 'punctuation';
          if (text.startsWith('"')) {
            kind = line
              .slice(match.index + text.length)
              .trimStart()
              .startsWith(':')
              ? 'key'
              : 'string';
          } else if (/^(true|false|null)$/.test(text)) {
            kind = 'boolean';
          } else if (/^-?\d/.test(text)) {
            kind = 'number';
          }
          tokens.push({ text, kind });
        }
        return tokens;
      }),
  );
}
