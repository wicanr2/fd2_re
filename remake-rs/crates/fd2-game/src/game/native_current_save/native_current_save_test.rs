//! Go 來源：`remake/cmd/fd2/native_current_save_test.go` 的三個測試都先經過 R8 的
//! `buildNativeCurrentSaveStored`（CONTINUE 目前戰況存檔），在 R8 移植（見 PORTING.md 的 F 組段落）。
//! 這裡只放 F 組函式 `replaceNativeCurrentSaveAtomic` 的補充測試。

use super::*;

/// F 組補充：無效輸入不碰檔案；成功時保留目標權限、不留暫存檔。
#[test]
fn replace_native_current_save_atomic_validates_and_preserves_mode() {
    let dir = crate::test_support::temp_dir("current-save-atomic");
    let path = dir.join("FD2.SAV");
    let path_str = path.to_string_lossy().into_owned();
    let old = fdsave::encode(&vec![1u8; fdsave::FILE_SIZE]).unwrap();
    std::fs::write(&path, &old).unwrap();
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
    }
    let err = replace_native_current_save_atomic(&path_str, &old[..5]).unwrap_err();
    assert_eq!(err.message(), "原版目前戰況存檔：原子寫入輸入無效");
    assert_eq!(
        replace_native_current_save_atomic("", &old).unwrap_err().message(),
        "原版目前戰況存檔：原子寫入輸入無效"
    );
    let missing = dir.join("missing.SAV").to_string_lossy().into_owned();
    assert!(
        replace_native_current_save_atomic(&missing, &old)
            .unwrap_err()
            .message()
            .starts_with("原版目前戰況存檔：讀取目標屬性：")
    );
    assert_eq!(std::fs::read(&path).unwrap(), old);

    let new = fdsave::encode(&vec![2u8; fdsave::FILE_SIZE]).unwrap();
    replace_native_current_save_atomic(&path_str, &new).unwrap();
    assert_eq!(std::fs::read(&path).unwrap(), new);
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        assert_eq!(std::fs::metadata(&path).unwrap().permissions().mode() & 0o777, 0o600);
    }
    for entry in std::fs::read_dir(&dir).unwrap() {
        let name = entry.unwrap().file_name().to_string_lossy().into_owned();
        assert!(!name.starts_with(".FD2.SAV.tmp-"), "暫存檔留下來了：{name}");
    }
}
