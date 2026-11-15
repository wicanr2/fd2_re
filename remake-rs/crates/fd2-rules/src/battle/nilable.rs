//! Go 切片與 map 的 nil 狀態。
//!
//! Go 的 `[]T` 與 `map[K]V` 有兩種「沒有元素」：nil 與非 nil 的空值。兩者行為幾乎相同，但
//! `encoding/json` 把 nil 寫成 `null`、空值寫成 `[]`／`{}`，`reflect.DeepEqual` 也把兩者當成不同。
//! Rust 的 `Vec`／`BTreeMap` 沒有 nil，直接對應會讓存檔位元組與 Go 不同（#319）。
//!
//! [`NilableVec`] 與 [`NilableMap`] 包住 `Vec`／`BTreeMap`，另外記住「是否為 nil」，以 `Deref` 讓
//! 呼叫端照舊當成 `Vec`／`BTreeMap` 使用。nil 與空值的界定照 Go：
//!
//! - 零值（`Default`）是 nil，對應 Go 結構欄位的零值。
//! - JSON 讀到 `null` 或缺欄是 nil，讀到 `[]`／`{}` 是非 nil 空值；寫出時 nil 寫 `null`，其餘照寫。
//! - 有元素就不是 nil。經 `DerefMut` 加入元素後即非 nil；之後再清空會回到 nil，Go 的
//!   `append(nil, x)[:0]` 卻是非 nil 空值。重製端沒有先加入再清空這些欄位的流程，列為已知差異。
//! - [`NilableVec::from_vec`]（與 `From<Vec<T>>`）是非 nil，對應 Go 的切片字面值與 `make`；
//!   [`NilableVec::appended`]（與 `FromIterator`）對應 `append([]T(nil), src...)`，沒有元素時是 nil。
//!   改寫整個欄位的呼叫端照 Go 原檔選用其中一個。
//! - `PartialEq` 照 `reflect.DeepEqual`：元素相同且 nil 狀態相同。和 `Vec`／`BTreeMap` 比較只比元素。

use std::collections::BTreeMap;
use std::ops::{Deref, DerefMut};

use serde::{Deserialize, Deserializer, Serialize, Serializer};

/// 記住 Go nil 狀態的 `Vec`。見模組文件。
#[derive(Debug, Clone)]
pub struct NilableVec<T> {
    items: Vec<T>,
    non_nil: bool,
}

impl<T> NilableVec<T> {
    /// Go 的 nil 切片。
    pub const fn nil() -> Self {
        Self { items: Vec::new(), non_nil: false }
    }

    /// Go 的非 nil 切片（字面值、`make`、讀到 JSON 陣列）。
    pub fn from_vec(items: Vec<T>) -> Self {
        Self { items, non_nil: true }
    }

    /// Go 的 `append([]T(nil), src...)`：沒有元素時是 nil。
    pub fn appended(items: Vec<T>) -> Self {
        Self { items, non_nil: false }
    }

    /// Go 的 `s == nil`。
    pub fn is_nil(&self) -> bool {
        !self.non_nil && self.items.is_empty()
    }

    /// 取出元素，丟掉 nil 狀態。
    pub fn into_vec(self) -> Vec<T> {
        self.items
    }
}

impl<T> Default for NilableVec<T> {
    fn default() -> Self {
        Self::nil()
    }
}

impl<T> Deref for NilableVec<T> {
    type Target = Vec<T>;
    fn deref(&self) -> &Vec<T> {
        &self.items
    }
}

impl<T> DerefMut for NilableVec<T> {
    fn deref_mut(&mut self) -> &mut Vec<T> {
        &mut self.items
    }
}

impl<T> From<Vec<T>> for NilableVec<T> {
    fn from(items: Vec<T>) -> Self {
        Self::from_vec(items)
    }
}

impl<T> FromIterator<T> for NilableVec<T> {
    fn from_iter<I: IntoIterator<Item = T>>(iter: I) -> Self {
        Self::appended(iter.into_iter().collect())
    }
}

impl<T> IntoIterator for NilableVec<T> {
    type Item = T;
    type IntoIter = std::vec::IntoIter<T>;
    fn into_iter(self) -> Self::IntoIter {
        self.items.into_iter()
    }
}

impl<'a, T> IntoIterator for &'a NilableVec<T> {
    type Item = &'a T;
    type IntoIter = std::slice::Iter<'a, T>;
    fn into_iter(self) -> Self::IntoIter {
        self.items.iter()
    }
}

impl<'a, T> IntoIterator for &'a mut NilableVec<T> {
    type Item = &'a mut T;
    type IntoIter = std::slice::IterMut<'a, T>;
    fn into_iter(self) -> Self::IntoIter {
        self.items.iter_mut()
    }
}

impl<T: PartialEq> PartialEq for NilableVec<T> {
    fn eq(&self, other: &Self) -> bool {
        self.is_nil() == other.is_nil() && self.items == other.items
    }
}

impl<T: PartialEq> PartialEq<Vec<T>> for NilableVec<T> {
    fn eq(&self, other: &Vec<T>) -> bool {
        &self.items == other
    }
}

impl<T: PartialEq> PartialEq<NilableVec<T>> for Vec<T> {
    fn eq(&self, other: &NilableVec<T>) -> bool {
        self == &other.items
    }
}

impl<T: PartialEq, const N: usize> PartialEq<[T; N]> for NilableVec<T> {
    fn eq(&self, other: &[T; N]) -> bool {
        self.items == other
    }
}

impl<T: Serialize> Serialize for NilableVec<T> {
    fn serialize<S: Serializer>(&self, s: S) -> Result<S::Ok, S::Error> {
        if self.is_nil() { s.serialize_none() } else { self.items.serialize(s) }
    }
}

impl<'de, T: Deserialize<'de>> Deserialize<'de> for NilableVec<T> {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        Ok(Option::<Vec<T>>::deserialize(d)?.map_or_else(Self::nil, Self::from_vec))
    }
}

/// 記住 Go nil 狀態的 `BTreeMap`。見模組文件。
#[derive(Debug, Clone)]
pub struct NilableMap<K, V> {
    items: BTreeMap<K, V>,
    non_nil: bool,
}

impl<K, V> NilableMap<K, V> {
    /// Go 的 nil map。
    pub const fn nil() -> Self {
        Self { items: BTreeMap::new(), non_nil: false }
    }

    /// Go 的非 nil map（字面值、`make`、讀到 JSON 物件）。
    pub fn from_map(items: BTreeMap<K, V>) -> Self {
        Self { items, non_nil: true }
    }

    /// Go 的 `m == nil`。
    pub fn is_nil(&self) -> bool {
        !self.non_nil && self.items.is_empty()
    }

    /// 取出元素，丟掉 nil 狀態。
    pub fn into_map(self) -> BTreeMap<K, V> {
        self.items
    }
}

impl<K, V> Default for NilableMap<K, V> {
    fn default() -> Self {
        Self::nil()
    }
}

impl<K, V> Deref for NilableMap<K, V> {
    type Target = BTreeMap<K, V>;
    fn deref(&self) -> &BTreeMap<K, V> {
        &self.items
    }
}

impl<K, V> DerefMut for NilableMap<K, V> {
    fn deref_mut(&mut self) -> &mut BTreeMap<K, V> {
        &mut self.items
    }
}

impl<K, V> From<BTreeMap<K, V>> for NilableMap<K, V> {
    fn from(items: BTreeMap<K, V>) -> Self {
        Self::from_map(items)
    }
}

impl<'a, K, V> IntoIterator for &'a NilableMap<K, V> {
    type Item = (&'a K, &'a V);
    type IntoIter = std::collections::btree_map::Iter<'a, K, V>;
    fn into_iter(self) -> Self::IntoIter {
        self.items.iter()
    }
}

impl<K: PartialEq, V: PartialEq> PartialEq for NilableMap<K, V> {
    fn eq(&self, other: &Self) -> bool {
        self.is_nil() == other.is_nil() && self.items == other.items
    }
}

impl<K: PartialEq, V: PartialEq> PartialEq<BTreeMap<K, V>> for NilableMap<K, V> {
    fn eq(&self, other: &BTreeMap<K, V>) -> bool {
        &self.items == other
    }
}

impl<K: Serialize, V: Serialize> Serialize for NilableMap<K, V> {
    fn serialize<S: Serializer>(&self, s: S) -> Result<S::Ok, S::Error> {
        if self.is_nil() { s.serialize_none() } else { s.collect_map(&self.items) }
    }
}

impl<'de, K: Deserialize<'de> + Ord, V: Deserialize<'de>> Deserialize<'de> for NilableMap<K, V> {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        Ok(Option::<BTreeMap<K, V>>::deserialize(d)?.map_or_else(Self::nil, Self::from_map))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nil_and_empty_round_trip_like_go_json() {
        let nil: NilableVec<i64> = serde_json::from_str("null").unwrap();
        let empty: NilableVec<i64> = serde_json::from_str("[]").unwrap();
        assert!(nil.is_nil());
        assert!(!empty.is_nil());
        assert_ne!(nil, empty);
        assert_eq!(serde_json::to_string(&nil).unwrap(), "null");
        assert_eq!(serde_json::to_string(&empty).unwrap(), "[]");
        assert_eq!(serde_json::to_string(&NilableVec::<i64>::default()).unwrap(), "null");
        assert_eq!(serde_json::to_string(&NilableVec::appended(Vec::<i64>::new())).unwrap(), "null");
        assert_eq!(serde_json::to_string(&NilableVec::appended(vec![1])).unwrap(), "[1]");

        let mut pushed = NilableVec::<i64>::nil();
        pushed.push(3);
        assert!(!pushed.is_nil());
        assert_eq!(pushed, vec![3]);

        let nil: NilableMap<String, bool> = serde_json::from_str("null").unwrap();
        let empty: NilableMap<String, bool> = serde_json::from_str("{}").unwrap();
        assert_eq!(serde_json::to_string(&nil).unwrap(), "null");
        assert_eq!(serde_json::to_string(&empty).unwrap(), "{}");
        assert_ne!(nil, empty);
    }
}
