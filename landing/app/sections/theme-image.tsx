import Image, { type ImageProps } from "next/image";

type Props = Omit<ImageProps, "src" | "preload" | "loading" | "priority"> & { srcLight: string; srcDark: string };

export function ThemeImage({ srcLight, srcDark, className = "", alt, ...rest }: Props) {
  return (
    <>
      <Image {...rest} alt={alt} src={srcLight} className={`img-light ${className}`} />
      <Image {...rest} alt={alt} src={srcDark} className={`img-dark ${className}`} />
    </>
  );
}
