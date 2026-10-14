import { renderToString } from 'react-dom/server';
import App from './App';
import { parseBrand } from './brand';

export function render(profile: unknown) {
  return renderToString(<App initialBrand={parseBrand(profile)} />);
}
