#!/usr/bin/env python3
"""CTranslate2 4.8 / Transformers 4.44 參數名稱相容橋接。"""
import argparse

from ctranslate2.converters import TransformersConverter


class CompatibleConverter(TransformersConverter):
    def load_model(self, model_class, model_name_or_path, **kwargs):
        # CTranslate2 4.8 使用新版 Transformers 的 dtype 名稱；4.44 仍名為 torch_dtype。
        if "dtype" in kwargs:
            kwargs["torch_dtype"] = kwargs.pop("dtype")
        return model_class.from_pretrained(model_name_or_path, **kwargs)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    CompatibleConverter(args.model).convert(args.output, quantization="int8", force=True)


if __name__ == "__main__":
    main()
