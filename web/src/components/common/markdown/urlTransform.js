import { defaultUrlTransform } from 'react-markdown';

const dataImageUrlPattern =
  /^data:image\/(?:png|jpe?g|webp|gif);base64,[a-z0-9+/=\s]+$/i;

export function markdownUrlTransform(value, key, node) {
  if (
    key === 'src' &&
    node?.tagName === 'img' &&
    dataImageUrlPattern.test(value)
  ) {
    return value;
  }

  return defaultUrlTransform(value);
}
