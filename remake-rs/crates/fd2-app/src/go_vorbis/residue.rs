//! jfreymuth/vorbis v1.0.2 的 `residue.go`。授權見 [`super`] 模組文件。
//!
//! 照 Go 保留兩個與 Vorbis I 規格不同的行為：`classifications` 以 `j*cs + partitionCount` 排列；
//! 殘差型別 2 只看第一個聲道的 `doNotDecode`（外層 `ch` 已改成 1）。

use super::VorbisError;
use super::codebook::{BitReader, Codebook};

pub(super) struct Residue {
    residue_type: u16,
    begin: u32,
    end: u32,
    partition_size: u32,
    classifications: u8,
    classbook: u8,
    #[allow(dead_code)]
    cascade: Vec<u8>,
    books: Vec<[i16; 8]>,
}

impl Residue {
    pub(super) fn read_from(r: &mut BitReader<'_>) -> Result<Residue, VorbisError> {
        let residue_type = r.read16(16);
        if residue_type > 2 {
            return Err(VorbisError::Decoding);
        }
        let begin = r.read32(24);
        let end = r.read32(24);
        let partition_size = r.read32(24) + 1;
        let classifications = r.read8(6) + 1;
        let classbook = r.read8(8);
        let mut cascade = vec![0u8; usize::from(classifications)];
        for c in &mut cascade {
            let mut high_bits: u8 = 0;
            let low_bits = r.read8(3);
            if r.read_bool() {
                high_bits = r.read8(5);
            }
            *c = high_bits.wrapping_mul(8).wrapping_add(low_bits);
        }
        let mut books = vec![[0i16; 8]; usize::from(classifications)];
        for (i, book) in books.iter_mut().enumerate() {
            for (j, slot) in book.iter_mut().enumerate() {
                if cascade[i] & (1 << j) != 0 {
                    *slot = i16::from(r.read8(8));
                } else {
                    *slot = -1;
                }
            }
        }
        Ok(Residue { residue_type, begin, end, partition_size, classifications, classbook, cascade, books })
    }

    /// `out` 是這個子映射的各聲道殘差向量（Go 的 `tmp`，每個都已切成 `n` 長）。
    pub(super) fn decode(
        &self,
        r: &mut BitReader<'_>,
        do_not_decode: &[bool],
        mut n: u32,
        books: &[Codebook],
        out: &mut [&mut [f32]],
    ) {
        let mut ch = do_not_decode.len() as u32;
        if self.residue_type == 2 {
            if !do_not_decode.iter().any(|&not| !not) {
                return;
            }
            n = n.wrapping_mul(ch);
            ch = 1;
        }
        let (mut begin, mut end) = (self.begin, self.end);
        if begin > n {
            begin = n;
        }
        if end > n {
            end = n;
        }
        let classbook = &books[usize::from(self.classbook)];
        let class_words_per_codeword = classbook.dimensions;
        let n_to_read = end.wrapping_sub(begin);
        let partitions_to_read = n_to_read / self.partition_size;

        if n_to_read == 0 {
            return;
        }
        let cs = partitions_to_read.wrapping_add(class_words_per_codeword);
        let mut classifications = vec![0u32; ch.wrapping_mul(cs) as usize];
        let nclass = u32::from(self.classifications);
        for pass in 0..8 {
            let mut partition_count: u32 = 0;
            while partition_count < partitions_to_read {
                if pass == 0 {
                    for j in 0..ch {
                        if !do_not_decode[j as usize] {
                            let mut temp = classbook.decode_scalar(r);
                            let mut i = class_words_per_codeword;
                            while i > 0 {
                                let at = j.wrapping_mul(cs).wrapping_add(i - 1).wrapping_add(partition_count);
                                classifications[at as usize] = temp % nclass;
                                temp /= nclass;
                                i -= 1;
                            }
                        }
                    }
                }
                let mut classword: u32 = 0;
                while classword < class_words_per_codeword && partition_count < partitions_to_read {
                    for j in 0..ch {
                        if do_not_decode[j as usize] {
                            continue;
                        }
                        let vqclass = classifications[j.wrapping_mul(cs).wrapping_add(partition_count) as usize];
                        let vqbook = self.books[vqclass as usize][pass];
                        if vqbook == -1 {
                            continue;
                        }
                        let book = &books[vqbook as usize];
                        let offset = begin.wrapping_add(partition_count.wrapping_mul(self.partition_size));
                        match self.residue_type {
                            0 => {
                                let step = self.partition_size / book.dimensions;
                                for i in 0..step {
                                    let tmp = book.decode_vector(r);
                                    for (k, &t) in tmp.iter().enumerate() {
                                        let at = offset.wrapping_add(i).wrapping_add((k as u32).wrapping_mul(step));
                                        out[j as usize][at as usize] += t;
                                    }
                                }
                            }
                            1 => {
                                let mut i: u32 = 0;
                                while i < self.partition_size {
                                    let tmp = book.decode_vector(r);
                                    for &t in tmp {
                                        out[j as usize][offset.wrapping_add(i) as usize] += t;
                                        i = i.wrapping_add(1);
                                    }
                                }
                            }
                            2 => {
                                let mut i: u32 = 0;
                                let ch = out.len() as u32;
                                while i < self.partition_size {
                                    let tmp = book.decode_vector(r);
                                    for &t in tmp {
                                        let at = offset.wrapping_add(i);
                                        out[(at % ch) as usize][(at / ch) as usize] += t;
                                        i = i.wrapping_add(1);
                                    }
                                }
                            }
                            _ => {}
                        }
                    }
                    partition_count += 1;
                    classword += 1;
                }
            }
        }
    }
}
