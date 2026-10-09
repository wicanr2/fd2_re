/* sub_165AC完整C候選。原始位址、型別寬度、五個全域槽與caller另存主收據。
 * 暫存與迴圈表示只用於匹配產碼，不推定作者宣告或硬體wall-clock。
 */
extern int dword_51A83,dword_53AB9,dword_53ABD,dword_53C67;
extern unsigned char *dword_53A81;
extern void *dword_53A18[5];
extern void sub_12CEA(int,int),j___delay(unsigned),sub_4E031(void);
extern void *sub_15E9E(void *,void *,int,int,int);
extern void sub_15E71(void *,void *,int),sub_4E96F(void *,int,int,void *,int,int);
extern void sub_168B6(void *,int,int,int,int,int);
extern void *malloc(unsigned);

#define CAPTURE(N) sub_4E96F(dword_53A18[N],310,86,(void *)0xa0000,offset,320)
#define DRAW(W,N) sub_168B6((void *)0xa0000,320,5,z,W,N)
#if defined(DIALOGUE_BACKUP_6) || defined(DIALOGUE_BACKUP_7) || defined(DIALOGUE_BACKUP_8)
typedef struct {int first,second,third;} RawThreeArguments;
void **sub_165AC(RawThreeArguments argument)
#define x argument.first
#define y argument.second
#define z argument.third
#elif defined(DIALOGUE_BACKUP_3) || defined(DIALOGUE_BACKUP_4) || defined(DIALOGUE_BACKUP_5)
void **sub_165AC(volatile int x,volatile int y,int z)
#else
void **sub_165AC(int x,int y,int z)
#endif
{
    int step,sum,offset;
    if(z){
        dword_51A83=0;sub_12CEA(x,y);dword_51A83=1;
        x=dword_53AB9*24+4;y=dword_53ABD*24+4;
        sum=dword_53AB9+dword_53ABD;
        if(sum){
            step=0;goto move_check;
move_next:
#ifdef DIALOGUE_BACKUP_1
            dword_53A18[0]=sub_15E9E(dword_53A81+*(short *)(dword_53A81+6),(void *)0xa0000,320,
                x-(x-5)*step/sum,y-(y-z)*step/sum);
#else
            {
                int px=x-(x-5)*step/sum;
                int py=y-(y-z)*step/sum;
#if defined(DIALOGUE_BACKUP_5) || defined(DIALOGUE_BACKUP_7)
                void *shape=dword_53A81+*(short *)(dword_53A81+6);
                dword_53A18[0]=sub_15E9E(shape,(void *)0xa0000,320,px,py);
#else
                dword_53A18[0]=sub_15E9E(dword_53A81+*(short *)(dword_53A81+6),(void *)0xa0000,320,px,py);
#endif
            }
#endif
            j___delay(10);sub_4E031();sub_15E71(dword_53A18[0],(void *)0xa0000,320);++step;
move_check:
            if(step<=sum)goto move_next;
        }
    }else if(dword_53C67==0x728)z=2;
    else if(dword_53C67==0x9017)z=112;
#if defined(DIALOGUE_BACKUP_2) || defined(DIALOGUE_BACKUP_4) || defined(DIALOGUE_BACKUP_5) || defined(DIALOGUE_BACKUP_7) || defined(DIALOGUE_BACKUP_8)
#ifdef DIALOGUE_BACKUP_7
    step=z*0;goto allocate_check;
#elif defined(DIALOGUE_BACKUP_4) || defined(DIALOGUE_BACKUP_5)
    step=z-z;goto allocate_check;
#else
    step=0;goto allocate_check;
#endif
allocate_next:
    dword_53A18[step]=malloc(26668);++step;
allocate_check:
    if(step<5)goto allocate_next;
#else
    for(step=0;step<5;++step)dword_53A18[step]=malloc(26668);
#endif
#if defined(DIALOGUE_BACKUP_3) || defined(DIALOGUE_BACKUP_4) || defined(DIALOGUE_BACKUP_5) || defined(DIALOGUE_BACKUP_6) || defined(DIALOGUE_BACKUP_7) || defined(DIALOGUE_BACKUP_8)
    sub_4E96F(dword_53A18[0],310,86,(void *)0xa0000,offset=z*320+5,320);
#else
    offset=z*320+5;CAPTURE(0);
#endif
    DRAW(4,2);j___delay(10);
    CAPTURE(1);DRAW(8,3);j___delay(10);
    CAPTURE(2);DRAW(12,4);j___delay(10);
    CAPTURE(3);DRAW(16,5);j___delay(10);
    CAPTURE(4);DRAW(19,5);sub_4E031();
    return dword_53A18;
}
